package rate

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sort"
	"strconv"
	"time"
)

const defaultChartEndpoint = "https://query2.finance.yahoo.com/v8/finance/chart/%s?interval=1d&period1=%s&period2=%s"

type Service struct {
	repo          Repository
	client        *http.Client
	chartEndpoint string
}

func NewService(repo Repository, opts ...Option) *Service {
	s := &Service{
		repo:          repo,
		client:        http.DefaultClient,
		chartEndpoint: defaultChartEndpoint,
	}
	for _, o := range opts {
		o(s)
	}
	return s
}

type Option func(*Service)

func WithClient(c *http.Client) Option {
	return func(s *Service) { s.client = c }
}

func WithChartEndpoint(ep string) Option {
	return func(s *Service) { s.chartEndpoint = ep }
}

// GetRates returns exchange rates for the given pair and date range.
// It checks the DB first and scrapes missing data if needed.
func (s *Service) GetRates(ctx context.Context, pair string, from, to time.Time) (map[time.Time]float64, error) {
	existing, err := s.repo.ExistingDates(ctx, pair, from, to)
	if err != nil {
		return nil, fmt.Errorf("check existing rates: %w", err)
	}

	totalDays := countWeekdays(from, to)
	coverageRatio := float64(len(existing)) / float64(max(totalDays, 1))

	if coverageRatio <= 0.8 || len(existing) == 0 {
		scraped, scrapeErr := s.fetchChart(ctx, pairToSymbol(pair), from, to)
		if scrapeErr != nil {
			slog.Error("failed to fetch exchange rates", "pair", pair, "error", scrapeErr)
			// Fall through — use whatever we have in DB
		} else {
			rates := make([]Rate, 0, len(scraped))
			for d, v := range scraped {
				if existing[d] {
					continue
				}
				rates = append(rates, Rate{Pair: pair, Date: d, Rate: v})
			}
			if len(rates) > 0 {
				n, saveErr := s.repo.SaveRates(ctx, rates)
				if saveErr != nil {
					return nil, fmt.Errorf("save rates: %w", saveErr)
				}
				slog.Info("saved exchange rates", "pair", pair, "new", n)
			}
		}
	}

	// Fetch all rates from DB
	dbRates, err := s.repo.ListRates(ctx, pair, from, to)
	if err != nil {
		return nil, fmt.Errorf("list rates: %w", err)
	}

	result := make(map[time.Time]float64, len(dbRates))
	for _, r := range dbRates {
		result[r.Date] = r.Rate
	}
	return result, nil
}

// chartResponse is the minimal Yahoo v8 chart API response structure.
type chartResponse struct {
	Chart struct {
		Result []struct {
			Timestamp  []int64 `json:"timestamp"`
			Indicators struct {
				Quote []struct {
					Close []float64 `json:"close"`
				} `json:"quote"`
			} `json:"indicators"`
		} `json:"result"`
		Error *struct {
			Code        string `json:"code"`
			Description string `json:"description"`
		} `json:"error"`
	} `json:"chart"`
}

// fetchChart fetches daily close prices from the Yahoo v8 chart API.
func (s *Service) fetchChart(ctx context.Context, symbol string, from, to time.Time) (map[time.Time]float64, error) {
	url := fmt.Sprintf(s.chartEndpoint, symbol,
		strconv.FormatInt(from.Unix(), 10),
		strconv.FormatInt(to.Unix(), 10))

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")

	res, err := s.client.Do(req) //nolint:gosec // URL built from internal config
	if err != nil {
		return nil, err
	}
	defer func() { _ = res.Body.Close() }()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("yahoo chart returned HTTP %d for %s", res.StatusCode, symbol)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	var cr chartResponse
	if err := json.Unmarshal(body, &cr); err != nil {
		return nil, fmt.Errorf("parse chart response: %w", err)
	}

	if cr.Chart.Error != nil {
		return nil, fmt.Errorf("yahoo chart error: %s: %s", cr.Chart.Error.Code, cr.Chart.Error.Description)
	}

	if len(cr.Chart.Result) == 0 {
		return nil, fmt.Errorf("yahoo chart returned no results for %s", symbol)
	}

	r := cr.Chart.Result[0]
	if len(r.Indicators.Quote) == 0 {
		return nil, fmt.Errorf("yahoo chart returned no quote data for %s", symbol)
	}

	timestamps := r.Timestamp
	closes := r.Indicators.Quote[0].Close

	rates := make(map[time.Time]float64, len(timestamps))
	for i, ts := range timestamps {
		if i >= len(closes) || closes[i] == 0 {
			continue
		}
		d := time.Unix(ts, 0).UTC().Truncate(24 * time.Hour)
		rates[d] = closes[i]
	}

	slog.Info("fetched exchange rates from chart API", "symbol", symbol, "count", len(rates))
	return rates, nil
}

// ForwardFill returns a rate for each date in dates, using the nearest prior rate
// when no exact match exists.
func ForwardFill(rates map[time.Time]float64, dates []time.Time) map[time.Time]float64 {
	// Collect and sort all known rate dates
	sortedDates := make([]time.Time, 0, len(rates))
	for d := range rates {
		sortedDates = append(sortedDates, d)
	}
	sort.Slice(sortedDates, func(i, j int) bool { return sortedDates[i].Before(sortedDates[j]) })

	result := make(map[time.Time]float64, len(dates))
	for _, d := range dates {
		if r, ok := rates[d]; ok {
			result[d] = r
			continue
		}
		// Find nearest prior date
		idx := sort.Search(len(sortedDates), func(i int) bool {
			return sortedDates[i].After(d)
		})
		if idx > 0 {
			result[d] = rates[sortedDates[idx-1]]
		}
	}
	return result
}

func pairToSymbol(pair string) string {
	if pair == PairUSDTRY {
		return "USDTRY=X"
	}
	return pair + "=X"
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
