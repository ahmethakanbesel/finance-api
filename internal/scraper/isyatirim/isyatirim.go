package isyatirim

import (
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
	defaultEndpoint = "https://www.isyatirim.com.tr/_Layouts/15/IsYatirim.Website/Common/ChartData.aspx/IndexHistoricalAll"
	dateFormat      = "20060102150405"
)

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

func WithEndpoint(ep string) Option {
	return func(s *Scraper) { s.endpoint = ep }
}

func (s *Scraper) Source() string { return "isyatirim" }

func (s *Scraper) NativeCurrency(_ string) string { return "TRY" }

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

	reqURL := fmt.Sprintf("%s?period=1440&from=%s&to=%s&endeks=%s",
		s.endpoint,
		from.Format(dateFormat),
		to.Format(dateFormat),
		symbol,
	)

	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
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
		return nil, fmt.Errorf("isyatirim returned HTTP %d for %s", res.StatusCode, symbol)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	var response struct {
		Data [][]json.Number `json:"data"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse isyatirim response: %w", err)
	}

	prices := make([]scraper.ScrapedPrice, 0, len(response.Data))
	for _, entry := range response.Data {
		if len(entry) < 2 {
			continue
		}

		tsMs, err := entry[0].Int64()
		if err != nil {
			continue
		}

		closePrice, err := entry[1].Float64()
		if err != nil {
			continue
		}

		date := time.Unix(tsMs/1000, (tsMs%1000)*1e6).UTC().Truncate(24 * time.Hour)
		prices = append(prices, scraper.ScrapedPrice{
			Date:       date,
			ClosePrice: closePrice,
		})
	}

	slog.Info("retrieved isyatirim data", "symbol", symbol,
		"from", from.Format("2006-01-02"), "to", to.Format("2006-01-02"),
		"count", len(prices))

	return prices, nil
}
