package tefas

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/ahmethakanbesel/finance-api/internal/scraper"
)

const (
	defaultEndpoint = "https://www.tefas.gov.tr/api/funds/fonFiyatBilgiGetir"
	dateFormat      = "2006-01-02"
)

var allowedPeriods = []int{1, 3, 6, 12, 36, 60}

type priceRecord struct {
	FundCode string  `json:"fonKodu"`
	FundName string  `json:"fonUnvan"`
	Date     string  `json:"tarih"`
	Price    float64 `json:"fiyat"`
}

type priceResponse struct {
	ErrorCode    *string       `json:"errorCode"`
	ErrorMessage *string       `json:"errorMessage"`
	ResultList   []priceRecord `json:"resultList"`
}

type Scraper struct {
	workers  int
	client   *http.Client
	endpoint string
}

func New(opts ...Option) *Scraper {
	s := &Scraper{
		workers:  5,
		client:   http.DefaultClient,
		endpoint: defaultEndpoint,
	}
	for _, o := range opts {
		o(s)
	}
	return s
}

type Option func(*Scraper)

func WithWorkers(n int) Option {
	return func(s *Scraper) { s.workers = n }
}

func WithClient(c *http.Client) Option {
	return func(s *Scraper) { s.client = c }
}

func WithEndpoint(url string) Option {
	return func(s *Scraper) { s.endpoint = url }
}

func (s *Scraper) Source() string { return "tefas" }

func (s *Scraper) NativeCurrency(_ string) string { return "TRY" }

func (s *Scraper) Scrape(ctx context.Context, symbol string, from, to time.Time) ([]scraper.ScrapedPrice, error) {
	if symbol == "" {
		return nil, fmt.Errorf("symbol cannot be empty")
	}
	if from.IsZero() {
		return nil, fmt.Errorf("start date cannot be empty")
	}
	now := time.Now()
	if to.IsZero() {
		to = now
	}
	if from.After(to) {
		return nil, fmt.Errorf("start date cannot be after end date")
	}

	periyod, covered := selectPeriod(from, now)
	if !covered {
		slog.Warn("tefas: requested range exceeds API limit; older data is unavailable",
			"fund", symbol, "maxMonths", periyod, "from", from.Format(dateFormat))
	}

	records, err := s.getFundData(ctx, symbol, periyod)
	if err != nil {
		return nil, err
	}

	prices := make([]scraper.ScrapedPrice, 0, len(records))
	for _, rec := range records {
		t, err := time.Parse(dateFormat, rec.Date)
		if err != nil || rec.Price < 0 {
			continue
		}
		t = t.UTC().Truncate(24 * time.Hour)
		// The API anchors the window to today, so clip to the requested range.
		if t.Before(from) || t.After(to) {
			continue
		}
		prices = append(prices, scraper.ScrapedPrice{
			Date:       t,
			ClosePrice: rec.Price,
		})
	}

	slog.Info("retrieved tefas data", "fund", symbol,
		"from", from.Format(dateFormat), "to", to.Format(dateFormat),
		"periyod", periyod, "count", len(prices))
	return prices, nil
}

func (s *Scraper) getFundData(ctx context.Context, fundCode string, periyod int) ([]priceRecord, error) {
	payload, err := json.Marshal(map[string]any{
		"fonKodu": fundCode,
		"dil":     "TR",
		"periyod": periyod,
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", s.endpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0")

	res, err := s.client.Do(req) //nolint:gosec // URL built from internal config
	if err != nil {
		return nil, err
	}
	defer func() { _ = res.Body.Close() }()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("tefas returned HTTP %d for %s", res.StatusCode, fundCode)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	var response priceResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse tefas response: %w", err)
	}
	if response.ErrorMessage != nil && *response.ErrorMessage != "" {
		return nil, fmt.Errorf("tefas error for %s: %s", fundCode, *response.ErrorMessage)
	}

	return response.ResultList, nil
}

func selectPeriod(from, now time.Time) (periyod int, covered bool) {
	for _, p := range allowedPeriods {
		if !now.AddDate(0, -p, 0).After(from) {
			return p, true
		}
	}
	return allowedPeriods[len(allowedPeriods)-1], false
}
