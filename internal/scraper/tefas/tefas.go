package tefas

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/ahmethakanbesel/finance-api/internal/scraper"
)

const (
	defaultBaseURL         = "https://www.tefas.gov.tr"
	defaultHistoryEndpoint = "https://www.tefas.gov.tr/api/DB/BindHistoryInfo"
	defaultReferer         = "http://www.tefas.gov.tr/TarihselVeriler.aspx"
	dateFormat             = "2006-01-02"
	chunkDays              = 60
)

type tefasPriceData struct {
	Timestamp  string  `json:"TARIH"`
	FundCode   string  `json:"FONKODU"`
	FundName   string  `json:"FONUNVAN"`
	Price      float64 `json:"FIYAT"`
	NumShares  float64 `json:"TEDPAYSAYISI"`
	NumPeople  float64 `json:"KISISAYISI"`
	TotalWorth float64 `json:"PORTFOYBUYUKLUK"`
}

type fundData struct {
	Draw            int              `json:"draw"`
	RecordsTotal    int              `json:"recordsTotal"`
	RecordsFiltered int              `json:"recordsFiltered"`
	Data            []tefasPriceData `json:"data"`
}

type Scraper struct {
	workers         int
	client          *http.Client
	historyEndpoint string
	baseURL         string
	referer         string
}

func New(opts ...Option) *Scraper {
	s := &Scraper{
		workers:         5,
		client:          http.DefaultClient,
		historyEndpoint: defaultHistoryEndpoint,
		baseURL:         defaultBaseURL,
		referer:         defaultReferer,
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

func WithHistoryEndpoint(url string) Option {
	return func(s *Scraper) { s.historyEndpoint = url }
}

func WithBaseURL(url string) Option {
	return func(s *Scraper) { s.baseURL = url }
}

func WithReferer(url string) Option {
	return func(s *Scraper) { s.referer = url }
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
	if to.IsZero() {
		to = time.Now()
	}
	if from.After(to) {
		return nil, fmt.Errorf("start date cannot be after end date")
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
			fd, err := s.getFundData(ctx, symbol, c.From, c.To)
			if err != nil {
				slog.Error("error retrieving tefas data", "fund", symbol,
					"startDate", c.From, "endDate", c.To, "error", err)
				return nil // continue other chunks
			}
			chunk := make([]scraper.ScrapedPrice, 0, len(fd.Data))
			for _, d := range fd.Data {
				t := parseTimestamp(d.Timestamp)
				if t.IsZero() || d.Price < 0 {
					continue
				}
				chunk = append(chunk, scraper.ScrapedPrice{
					Date:       t,
					ClosePrice: d.Price,
				})
			}
			results[i] = result{prices: chunk}
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

func (s *Scraper) getFundData(ctx context.Context, fundCode string, startDate, endDate time.Time) (*fundData, error) {
	params := url.Values{}
	params.Add("fontip", "YAT")
	params.Add("fonkod", fundCode)
	params.Add("bastarih", startDate.Format(dateFormat))
	params.Add("bittarih", endDate.Format(dateFormat))

	req, err := http.NewRequestWithContext(ctx, "POST", s.historyEndpoint, strings.NewReader(params.Encode()))
	if err != nil {
		return nil, err
	}

	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Header.Set("Origin", s.baseURL)
	req.Header.Set("Referer", s.referer)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	res, err := s.client.Do(req) //nolint:gosec // URL built from internal config
	if err != nil {
		return nil, err
	}
	defer func() { _ = res.Body.Close() }()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	response := &fundData{}
	if err := json.Unmarshal(body, response); err != nil {
		return nil, err
	}

	slog.Info("retrieved tefas data", "fund", fundCode, "startDate", startDate.Format(dateFormat), "endDate", endDate.Format(dateFormat))
	return response, nil
}

func parseTimestamp(timestamp string) time.Time {
	ms, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return time.Time{}
	}
	return time.Unix(ms/1000, (ms%1000)*1e6).UTC().Truncate(24 * time.Hour)
}
