package isyatirim

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestScrape(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("endeks") != "ALTINS1" {
			t.Errorf("expected endeks=ALTINS1, got %s", q.Get("endeks"))
		}
		if q.Get("period") != "1440" {
			t.Errorf("expected period=1440, got %s", q.Get("period"))
		}

		resp := map[string]any{
			"data": [][]any{
				{1735772400000.0, 36.65},
				{1735858800000.0, 36.74},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	s := New(
		WithWorkers(1),
		WithClient(ts.Client()),
		WithEndpoint(ts.URL),
	)

	from := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2025, 1, 31, 0, 0, 0, 0, time.UTC)

	prices, err := s.Scrape(context.Background(), "ALTINS1", from, to)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(prices) != 2 {
		t.Fatalf("expected 2 prices, got %d", len(prices))
	}

	if prices[0].ClosePrice != 36.65 {
		t.Errorf("expected close 36.65, got %f", prices[0].ClosePrice)
	}
	if prices[1].ClosePrice != 36.74 {
		t.Errorf("expected close 36.74, got %f", prices[1].ClosePrice)
	}
}

func TestScrape_EmptyData(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"data": [][]any{}})
	}))
	defer ts.Close()

	s := New(WithEndpoint(ts.URL), WithClient(ts.Client()))

	from := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2025, 1, 31, 0, 0, 0, 0, time.UTC)

	prices, err := s.Scrape(context.Background(), "ALTINS1", from, to)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(prices) != 0 {
		t.Errorf("expected 0 prices, got %d", len(prices))
	}
}

func TestScrape_ErrorResponse(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	s := New(WithEndpoint(ts.URL), WithClient(ts.Client()))

	from := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2025, 1, 31, 0, 0, 0, 0, time.UTC)

	_, err := s.Scrape(context.Background(), "ALTINS1", from, to)
	if err == nil {
		t.Fatal("expected error for HTTP 500")
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
	if s.Source() != "isyatirim" {
		t.Errorf("expected source 'isyatirim', got '%s'", s.Source())
	}
}

func TestNativeCurrency(t *testing.T) {
	s := New()
	if s.NativeCurrency("ALTINS1") != "TRY" {
		t.Errorf("expected TRY, got %s", s.NativeCurrency("ALTINS1"))
	}
}
