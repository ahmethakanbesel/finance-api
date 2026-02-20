// Package yahoo implements a scraper for Yahoo Finance historical price data.
// It uses the v8 chart API with cookie + crumb authentication, matching the
// approach used by the yfinance Python library.
package yahoo

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/ahmethakanbesel/finance-api/internal/scraper"
)

const (
	defaultChartEndpoint = "https://query2.finance.yahoo.com/v8/finance/chart"
	defaultCookieURL     = "https://fc.yahoo.com"
	defaultCrumbURL      = "https://query1.finance.yahoo.com/v1/test/getcrumb"
	dateFormat           = "2006-01-02"
	chunkDays            = 1250
	userAgent            = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"
)

// Scraper fetches historical price data from Yahoo Finance.
type Scraper struct {
	workers       int
	client        *http.Client
	chartEndpoint string
	cookieURL     string
	crumbURL      string

	mu    sync.Mutex
	crumb string
}

// New creates a Scraper with the given options applied.
func New(opts ...Option) *Scraper {
	jar, _ := cookiejar.New(nil)
	s := &Scraper{
		workers:       5,
		client:        &http.Client{Jar: jar},
		chartEndpoint: defaultChartEndpoint,
		cookieURL:     defaultCookieURL,
		crumbURL:      defaultCrumbURL,
	}
	for _, o := range opts {
		o(s)
	}
	return s
}

// Option configures a Scraper.
type Option func(*Scraper)

// WithWorkers sets the worker concurrency for parallel chunk fetching.
func WithWorkers(n int) Option {
	return func(s *Scraper) { s.workers = n }
}

// WithClient sets the HTTP client. The client should have a cookie jar.
func WithClient(c *http.Client) Option {
	return func(s *Scraper) { s.client = c }
}

// WithChartEndpoint overrides the default chart API endpoint.
func WithChartEndpoint(ep string) Option {
	return func(s *Scraper) { s.chartEndpoint = ep }
}

// WithCookieURL overrides the URL used to obtain the session cookie.
func WithCookieURL(u string) Option {
	return func(s *Scraper) { s.cookieURL = u }
}

// WithCrumbURL overrides the URL used to obtain the crumb token.
func WithCrumbURL(u string) Option {
	return func(s *Scraper) { s.crumbURL = u }
}

// Source returns the scraper identifier.
func (s *Scraper) Source() string { return "yahoo" }

// NativeCurrency returns the currency that prices are denominated in.
// Symbols ending in ".IS" (Istanbul Stock Exchange) are in TRY; others in USD.
func (s *Scraper) NativeCurrency(symbol string) string {
	if strings.HasSuffix(symbol, ".IS") {
		return "TRY"
	}
	return "USD"
}

// chartResponse represents the Yahoo Finance v8 chart API response.
type chartResponse struct {
	Chart struct {
		Result []struct {
			Timestamp  []int64 `json:"timestamp"`
			Indicators struct {
				Quote []struct {
					Close []any `json:"close"`
				} `json:"quote"`
			} `json:"indicators"`
		} `json:"result"`
		Error *struct {
			Code        string `json:"code"`
			Description string `json:"description"`
		} `json:"error"`
	} `json:"chart"`
}

// Scrape fetches daily close prices for the given symbol and date range.
func (s *Scraper) Scrape(ctx context.Context, symbol string, from, to time.Time) ([]scraper.ScrapedPrice, error) {
	if symbol == "" {
		return nil, fmt.Errorf("symbol cannot be empty")
	}
	if from.IsZero() {
		return nil, fmt.Errorf("start date cannot be empty")
	}
	if to.IsZero() {
		to = time.Now()
	}
	if from.After(to) {
		return nil, fmt.Errorf("start date cannot be after end date")
	}

	// Ensure we have a valid crumb before starting parallel fetches.
	if err := s.ensureCrumb(ctx); err != nil {
		return nil, fmt.Errorf("yahoo auth: %w", err)
	}

	chunks := scraper.SplitDateRange(from, to, chunkDays)

	type result struct {
		prices []scraper.ScrapedPrice
	}
	results := make([]result, len(chunks))

	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(s.workers)

	for i, c := range chunks {
		g.Go(func() error {
			prices, err := s.fetchChart(ctx, symbol, c.From, c.To)
			if err != nil {
				slog.Error("error retrieving yahoo data", "symbol", symbol,
					"startDate", c.From.Format(dateFormat), "endDate", c.To.Format(dateFormat), "error", err)
				return nil
			}
			results[i] = result{prices: prices}
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	var all []scraper.ScrapedPrice
	for _, r := range results {
		all = append(all, r.prices...)
	}
	return all, nil
}

// ensureCrumb fetches a session cookie and crumb token if not already cached.
func (s *Scraper) ensureCrumb(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.crumb != "" {
		return nil
	}

	// Step 1: GET fc.yahoo.com to obtain a session cookie.
	cookieReq, err := http.NewRequestWithContext(ctx, "GET", s.cookieURL, nil)
	if err != nil {
		return fmt.Errorf("build cookie request: %w", err)
	}
	cookieReq.Header.Set("User-Agent", userAgent)

	cookieRes, err := s.client.Do(cookieReq) //nolint:gosec // URL from internal config
	if err != nil {
		return fmt.Errorf("fetch cookie: %w", err)
	}
	_ = cookieRes.Body.Close()

	// Step 2: GET crumb endpoint (cookie is sent automatically via jar).
	crumbReq, err := http.NewRequestWithContext(ctx, "GET", s.crumbURL, nil)
	if err != nil {
		return fmt.Errorf("build crumb request: %w", err)
	}
	crumbReq.Header.Set("User-Agent", userAgent)

	crumbRes, err := s.client.Do(crumbReq) //nolint:gosec // URL from internal config
	if err != nil {
		return fmt.Errorf("fetch crumb: %w", err)
	}
	defer func() { _ = crumbRes.Body.Close() }()

	if crumbRes.StatusCode != http.StatusOK {
		return fmt.Errorf("crumb endpoint returned HTTP %d", crumbRes.StatusCode)
	}

	body, err := io.ReadAll(crumbRes.Body)
	if err != nil {
		return fmt.Errorf("read crumb: %w", err)
	}

	crumb := strings.TrimSpace(string(body))
	if crumb == "" {
		return fmt.Errorf("empty crumb received")
	}

	s.crumb = crumb
	slog.Info("yahoo: obtained crumb", "crumb_len", len(crumb))
	return nil
}

// fetchChart fetches chart data for a single date range chunk.
func (s *Scraper) fetchChart(ctx context.Context, symbol string, from, to time.Time) ([]scraper.ScrapedPrice, error) {
	s.mu.Lock()
	crumb := s.crumb
	s.mu.Unlock()

	reqURL := fmt.Sprintf("%s/%s?period1=%s&period2=%s&interval=1d&events=div%%2Csplits&crumb=%s",
		s.chartEndpoint,
		symbol,
		strconv.FormatInt(from.Unix(), 10),
		strconv.FormatInt(to.Unix(), 10),
		crumb,
	)

	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)

	res, err := s.client.Do(req) //nolint:gosec // URL built from internal config
	if err != nil {
		return nil, err
	}
	defer func() { _ = res.Body.Close() }()

	if res.StatusCode != http.StatusOK {
		// Invalidate crumb on auth errors so next Scrape retries auth.
		if res.StatusCode == http.StatusUnauthorized || res.StatusCode == http.StatusForbidden {
			s.mu.Lock()
			s.crumb = ""
			s.mu.Unlock()
		}
		return nil, fmt.Errorf("yahoo returned HTTP %d for %s", res.StatusCode, symbol)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	var resp chartResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse yahoo response: %w", err)
	}

	if resp.Chart.Error != nil {
		return nil, fmt.Errorf("yahoo chart error: %s: %s", resp.Chart.Error.Code, resp.Chart.Error.Description)
	}

	if len(resp.Chart.Result) == 0 {
		return nil, nil
	}

	result := resp.Chart.Result[0]
	if len(result.Indicators.Quote) == 0 {
		return nil, nil
	}

	closes := result.Indicators.Quote[0].Close
	n := min(len(result.Timestamp), len(closes))
	prices := make([]scraper.ScrapedPrice, 0, n)
	for i := range n {
		closeVal, ok := toFloat64(closes[i])
		if !ok {
			continue
		}
		date := time.Unix(result.Timestamp[i], 0).UTC().Truncate(24 * time.Hour)
		prices = append(prices, scraper.ScrapedPrice{
			Date:       date,
			ClosePrice: closeVal,
		})
	}

	slog.Info("retrieved yahoo data", "symbol", symbol,
		"from", from.Format(dateFormat), "to", to.Format(dateFormat),
		"count", len(prices))

	return prices, nil
}

// toFloat64 converts a JSON number (which may be float64 or json.Number) to float64.
// Returns false for nil values (Yahoo uses null for missing data points).
func toFloat64(v any) (float64, bool) {
	switch val := v.(type) {
	case float64:
		return val, true
	case json.Number:
		f, err := val.Float64()
		return f, err == nil
	default:
		return 0, false
	}
}
