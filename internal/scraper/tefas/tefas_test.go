package tefas

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestScrape(t *testing.T) {
	var gotBody map[string]any
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("unexpected content type: %s", r.Header.Get("Content-Type"))
		}
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &gotBody)

		resp := priceResponse{
			ResultList: []priceRecord{
				{FundCode: "YAC", FundName: "Test", Date: "2024-01-15", Price: 1.23},
				{FundCode: "YAC", FundName: "Test", Date: "2024-01-16", Price: 1.24},
				// Outside the requested window; must be filtered out.
				{FundCode: "YAC", FundName: "Test", Date: "2023-12-31", Price: 9.99},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	s := New(
		WithClient(ts.Client()),
		WithEndpoint(ts.URL),
	)

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC)

	prices, err := s.Scrape(context.Background(), "YAC", from, to)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if gotBody["fonKodu"] != "YAC" {
		t.Errorf("expected fonKodu YAC in request body, got %v", gotBody["fonKodu"])
	}
	if _, ok := gotBody["periyod"]; !ok {
		t.Errorf("expected periyod in request body, got %v", gotBody)
	}

	if len(prices) != 2 {
		t.Fatalf("expected 2 prices (window-filtered), got %d", len(prices))
	}
	if prices[0].ClosePrice != 1.23 {
		t.Errorf("expected price 1.23, got %f", prices[0].ClosePrice)
	}
	if !prices[0].Date.Equal(time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("unexpected date: %v", prices[0].Date)
	}
}

func TestScrape_APIError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		msg := "Sistem Hatası!!"
		_ = json.NewEncoder(w).Encode(priceResponse{ErrorMessage: &msg})
	}))
	defer ts.Close()

	s := New(WithClient(ts.Client()), WithEndpoint(ts.URL))
	_, err := s.Scrape(context.Background(), "YAC",
		time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC))
	if err == nil {
		t.Fatal("expected error when API returns errorMessage")
	}
}

func TestScrape_HTTPError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	s := New(WithClient(ts.Client()), WithEndpoint(ts.URL))
	_, err := s.Scrape(context.Background(), "YAC",
		time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC))
	if err == nil {
		t.Fatal("expected error on non-200 status")
	}
}

func TestScrape_EmptySymbol(t *testing.T) {
	s := New()
	_, err := s.Scrape(context.Background(), "", time.Now(), time.Now())
	if err == nil {
		t.Fatal("expected error for empty symbol")
	}
}

func TestSelectPeriod(t *testing.T) {
	now := time.Date(2024, 3, 20, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		from        time.Time
		wantPeriyod int
		wantCovered bool
	}{
		{time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC), 1, true},   // ~3 weeks
		{time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC), 3, true},  // ~2.3 months -> bucket 3
		{time.Date(2023, 9, 20, 0, 0, 0, 0, time.UTC), 6, true},  // exactly 6 months
		{time.Date(2023, 3, 20, 0, 0, 0, 0, time.UTC), 12, true}, // exactly 1 year
		{time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC), 60, true},  // ~4y -> bucket 60
		{time.Date(2010, 1, 1, 0, 0, 0, 0, time.UTC), 60, false}, // beyond 5y -> capped
		{now.AddDate(0, 0, 1), 1, true},                          // future from -> smallest
	}
	for _, tt := range tests {
		p, covered := selectPeriod(tt.from, now)
		if p != tt.wantPeriyod || covered != tt.wantCovered {
			t.Errorf("selectPeriod(%s) = (%d, %v), want (%d, %v)",
				tt.from.Format(dateFormat), p, covered, tt.wantPeriyod, tt.wantCovered)
		}
	}
}
