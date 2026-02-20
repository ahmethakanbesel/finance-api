package yahoo

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// newTestServer returns a mock Yahoo Finance server that serves cookie, crumb,
// and chart endpoints, along with a Scraper configured to use it.
func newTestServer(t *testing.T, chartData chartResponse) (*httptest.Server, *Scraper) {
	t.Helper()

	mux := http.NewServeMux()

	// Cookie endpoint — just set a cookie.
	mux.HandleFunc("/cookie", func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{Name: "A3", Value: "test-session"})
		w.WriteHeader(http.StatusOK)
	})

	// Crumb endpoint — return a crumb string.
	mux.HandleFunc("/crumb", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("test-crumb-123"))
	})

	// Chart endpoint — return the provided chart data.
	mux.HandleFunc("/chart/", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("crumb") != "test-crumb-123" {
			t.Errorf("expected crumb=test-crumb-123, got %s", q.Get("crumb"))
		}
		if q.Get("interval") != "1d" {
			t.Errorf("expected interval=1d, got %s", q.Get("interval"))
		}
		_ = json.NewEncoder(w).Encode(chartData)
	})

	ts := httptest.NewServer(mux)

	s := New(
		WithWorkers(1),
		WithClient(ts.Client()),
		WithChartEndpoint(ts.URL+"/chart"),
		WithCookieURL(ts.URL+"/cookie"),
		WithCrumbURL(ts.URL+"/crumb"),
	)

	return ts, s
}

func TestScrape(t *testing.T) {
	resp := chartResponse{}
	resp.Chart.Result = []struct {
		Timestamp  []int64 `json:"timestamp"`
		Indicators struct {
			Quote []struct {
				Close []any `json:"close"`
			} `json:"quote"`
		} `json:"indicators"`
	}{
		{
			Timestamp: []int64{1704153600, 1704240000},
			Indicators: struct {
				Quote []struct {
					Close []any `json:"close"`
				} `json:"quote"`
			}{
				Quote: []struct {
					Close []any `json:"close"`
				}{
					{Close: []any{185.01, 184.25}},
				},
			},
		},
	}

	ts, s := newTestServer(t, resp)
	defer ts.Close()

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC)

	prices, err := s.Scrape(context.Background(), "AAPL", from, to)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(prices) != 2 {
		t.Fatalf("expected 2 prices, got %d", len(prices))
	}

	if prices[0].ClosePrice != 185.01 {
		t.Errorf("expected close 185.01, got %f", prices[0].ClosePrice)
	}
	if prices[1].ClosePrice != 184.25 {
		t.Errorf("expected close 184.25, got %f", prices[1].ClosePrice)
	}
}

func TestScrape_NullCloseValues(t *testing.T) {
	resp := chartResponse{}
	resp.Chart.Result = []struct {
		Timestamp  []int64 `json:"timestamp"`
		Indicators struct {
			Quote []struct {
				Close []any `json:"close"`
			} `json:"quote"`
		} `json:"indicators"`
	}{
		{
			Timestamp: []int64{1704153600, 1704240000, 1704326400},
			Indicators: struct {
				Quote []struct {
					Close []any `json:"close"`
				} `json:"quote"`
			}{
				Quote: []struct {
					Close []any `json:"close"`
				}{
					{Close: []any{185.01, nil, 184.25}},
				},
			},
		},
	}

	ts, s := newTestServer(t, resp)
	defer ts.Close()

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC)

	prices, err := s.Scrape(context.Background(), "AAPL", from, to)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(prices) != 2 {
		t.Fatalf("expected 2 prices (nil skipped), got %d", len(prices))
	}
}

func TestScrape_EmptyResult(t *testing.T) {
	resp := chartResponse{}
	ts, s := newTestServer(t, resp)
	defer ts.Close()

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC)

	prices, err := s.Scrape(context.Background(), "INVALID", from, to)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if prices != nil {
		t.Errorf("expected nil prices, got %d", len(prices))
	}
}

func TestScrape_ChartError(t *testing.T) {
	resp := chartResponse{}
	resp.Chart.Error = &struct {
		Code        string `json:"code"`
		Description string `json:"description"`
	}{Code: "Not Found", Description: "No data found"}

	ts, s := newTestServer(t, resp)
	defer ts.Close()

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC)

	// Chart errors are logged but not propagated (consistent with chunk error
	// handling — partial failures don't fail the entire scrape).
	prices, err := s.Scrape(context.Background(), "INVALID", from, to)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(prices) != 0 {
		t.Errorf("expected 0 prices for chart error, got %d", len(prices))
	}
}

func TestScrape_EmptySymbol(t *testing.T) {
	s := New()
	_, err := s.Scrape(context.Background(), "", time.Now(), time.Now())
	if err == nil {
		t.Fatal("expected error for empty symbol")
	}
}

func TestSource(t *testing.T) {
	s := New()
	if s.Source() != "yahoo" {
		t.Errorf("expected source 'yahoo', got '%s'", s.Source())
	}
}

func TestNativeCurrency(t *testing.T) {
	s := New()

	tests := []struct {
		symbol string
		want   string
	}{
		{"AAPL", "USD"},
		{"THYAO.IS", "TRY"},
		{"MSFT", "USD"},
	}

	for _, tt := range tests {
		if got := s.NativeCurrency(tt.symbol); got != tt.want {
			t.Errorf("NativeCurrency(%q) = %q, want %q", tt.symbol, got, tt.want)
		}
	}
}
