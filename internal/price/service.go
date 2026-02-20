package price

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/ahmethakanbesel/finance-api/internal/job"
	"github.com/ahmethakanbesel/finance-api/internal/rate"
	"github.com/ahmethakanbesel/finance-api/internal/scraper"
)

type Service struct {
	priceRepo Repository
	jobRepo   job.Repository
	registry  *scraper.Registry
	rateSvc   *rate.Service
	notify    func() // optional: wake worker pool
}

func NewService(priceRepo Repository, jobRepo job.Repository, registry *scraper.Registry, rateSvc *rate.Service) *Service {
	return &Service{
		priceRepo: priceRepo,
		jobRepo:   jobRepo,
		registry:  registry,
		rateSvc:   rateSvc,
	}
}

// SetNotify sets a callback invoked when a new pending job is created.
func (s *Service) SetNotify(fn func()) { s.notify = fn }

func (s *Service) ListSources() []string {
	return s.registry.Sources()
}

func (s *Service) GetPrices(ctx context.Context, req GetPricesRequest) (*GetPricesResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	endDate := req.EndDate
	if endDate.IsZero() {
		endDate = time.Now().Truncate(24 * time.Hour)
	}

	// Get the scraper to determine native currency
	sc, err := s.registry.Get(string(req.Source))
	if err != nil {
		return nil, err
	}
	nativeCurrency := Currency(sc.NativeCurrency(req.Symbol))

	// Check existing dates in DB (no currency filter — prices stored in native currency)
	existing, err := s.priceRepo.ExistingDates(ctx, req.Source, req.Symbol, req.StartDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("check existing dates: %w", err)
	}

	// Count expected business days (rough heuristic: weekdays)
	totalDays := countWeekdays(req.StartDate, endDate)
	coverageRatio := float64(len(existing)) / float64(max(totalDays, 1))

	var j *job.Job

	// If we don't have good coverage, queue a scraping job
	if coverageRatio <= 0.8 || len(existing) == 0 {
		// Dedup: check if there's already an active job for this range
		dateFormat := "2006-01-02"
		active, findErr := s.jobRepo.FindActive(ctx, string(req.Source), req.Symbol,
			req.StartDate.Format(dateFormat), endDate.Format(dateFormat))
		if findErr != nil {
			return nil, fmt.Errorf("find active job: %w", findErr)
		}

		if active != nil {
			j = active
		} else {
			// Create pending job for the worker pool to pick up
			j = &job.Job{
				Source:    string(req.Source),
				Symbol:    req.Symbol,
				StartDate: req.StartDate,
				EndDate:   endDate,
				Status:    job.StatusPending,
			}
			if createErr := s.jobRepo.Create(ctx, j); createErr != nil {
				return nil, fmt.Errorf("create job: %w", createErr)
			}
			if s.notify != nil {
				s.notify()
			}
		}
	}

	// Fetch all prices from DB (native currency)
	prices, err := s.priceRepo.ListPrices(ctx, req.Source, req.Symbol, req.StartDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("list prices: %w", err)
	}

	// Build PricePoints with conversion
	points, err := s.convertPrices(ctx, prices, nativeCurrency, req.Currency, req.StartDate, endDate)
	if err != nil {
		return nil, err
	}

	return &GetPricesResponse{Prices: points, Job: j}, nil
}

// Process implements job.Processor. Called by the worker pool with a claimed
// (running) job. It scrapes prices, saves them, and marks the job completed or failed.
func (s *Service) Process(ctx context.Context, j *job.Job) error {
	sc, err := s.registry.Get(j.Source)
	if err != nil {
		return s.failJob(ctx, j, err)
	}

	nativeCurrency := Currency(sc.NativeCurrency(j.Symbol))

	// Check existing dates to avoid duplicates
	existing, err := s.priceRepo.ExistingDates(ctx, Source(j.Source), j.Symbol, j.StartDate, j.EndDate)
	if err != nil {
		return s.failJob(ctx, j, fmt.Errorf("check existing dates: %w", err))
	}

	// Scrape
	scraped, err := sc.Scrape(ctx, j.Symbol, j.StartDate, j.EndDate)
	if err != nil {
		return s.failJob(ctx, j, fmt.Errorf("scrape: %w", err))
	}

	// Filter out already-existing dates
	newPrices := make([]Price, 0, len(scraped))
	for _, sp := range scraped {
		if existing[sp.Date] {
			continue
		}
		newPrices = append(newPrices, Price{
			Source:     Source(j.Source),
			Symbol:     j.Symbol,
			Date:       sp.Date,
			ClosePrice: sp.ClosePrice,
			Currency:   nativeCurrency,
		})
	}

	// Save
	n, err := s.priceRepo.SavePrices(ctx, newPrices)
	if err != nil {
		return s.failJob(ctx, j, fmt.Errorf("save prices: %w", err))
	}

	slog.Info("saved prices", "source", j.Source, "symbol", j.Symbol, "new", n, "total_scraped", len(scraped))

	// Mark completed
	j.Status = job.StatusCompleted
	j.RecordsCount = n
	_ = s.jobRepo.Update(ctx, j)
	return nil
}

func (s *Service) failJob(ctx context.Context, j *job.Job, err error) error {
	j.Status = job.StatusFailed
	j.Error = err.Error()
	_ = s.jobRepo.Update(ctx, j)
	return err
}

func (s *Service) convertPrices(ctx context.Context, prices []Price, nativeCurrency, requestedCurrency Currency, from, to time.Time) ([]PricePoint, error) {
	needConversion := nativeCurrency != requestedCurrency

	var rates map[time.Time]float64
	if needConversion {
		if s.rateSvc == nil {
			return nil, fmt.Errorf("currency conversion unavailable: rate service not configured")
		}
		var err error
		rates, err = s.rateSvc.GetRates(ctx, rate.PairUSDTRY, from, to)
		if err != nil {
			return nil, fmt.Errorf("get exchange rates: %w", err)
		}
		if len(rates) == 0 {
			return nil, fmt.Errorf("no exchange rates available for %s in the requested date range", rate.PairUSDTRY)
		}

		// Forward-fill rates for all price dates
		priceDates := make([]time.Time, len(prices))
		for i, p := range prices {
			priceDates[i] = p.Date
		}
		rates = rate.ForwardFill(rates, priceDates)
	}

	points := make([]PricePoint, len(prices))
	for i, p := range prices {
		pp := PricePoint{
			Symbol:         p.Symbol,
			Date:           p.Date,
			NativePrice:    p.ClosePrice,
			NativeCurrency: nativeCurrency,
			Currency:       requestedCurrency,
			Source:         p.Source,
			Rate:           1.0,
			ClosePrice:     p.ClosePrice,
		}

		if needConversion {
			r, ok := rates[p.Date]
			if !ok || r <= 0 {
				return nil, fmt.Errorf("missing exchange rate for %s on %s", rate.PairUSDTRY, p.Date.Format("2006-01-02"))
			}
			pp.Rate = r
			pp.ClosePrice = convert(p.ClosePrice, nativeCurrency, requestedCurrency, r)
		}

		points[i] = pp
	}

	return points, nil
}

// convert applies exchange rate conversion.
// The rate is always USDTRY (how many TRY per 1 USD).
func convert(price float64, from, to Currency, usdtryRate float64) float64 {
	switch {
	case from == CurrencyTRY && to == CurrencyUSD:
		return price / usdtryRate
	case from == CurrencyUSD && to == CurrencyTRY:
		return price * usdtryRate
	default:
		return price
	}
}

func countWeekdays(from, to time.Time) int {
	count := 0
	for d := from; !d.After(to); d = d.AddDate(0, 0, 1) {
		wd := d.Weekday()
		if wd != time.Saturday && wd != time.Sunday {
			count++
		}
	}
	return count
}
