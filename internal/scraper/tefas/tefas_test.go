package tefas

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
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/x-www-form-urlencoded" {
			t.Errorf("unexpected content type: %s", r.Header.Get("Content-Type"))
		}

		resp := fundData{
			RecordsTotal: 2,
			Data: []tefasPriceData{
				{Timestamp: "1704067200000", FundCode: "YAC", Price: 1.23},
				{Timestamp: "1704153600000", FundCode: "YAC", Price: 1.24},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	s := New(
		WithWorkers(1),
		WithClient(ts.Client()),
		WithHistoryEndpoint(ts.URL),
		WithBaseURL(ts.URL),
		WithReferer(ts.URL),
	)

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC)

	prices, err := s.Scrape(context.Background(), "YAC", from, to)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(prices) != 2 {
		t.Fatalf("expected 2 prices, got %d", len(prices))
	}

	if prices[0].ClosePrice != 1.23 {
		t.Errorf("expected price 1.23, got %f", prices[0].ClosePrice)
	}
}

func TestScrape_EmptySymbol(t *testing.T) {
	s := New()
	_, err := s.Scrape(context.Background(), "", time.Now(), time.Now())
	if err == nil {
		t.Fatal("expected error for empty symbol")
	}
}

func TestParseTimestamp(t *testing.T) {
	// 2024-01-01 00:00:00 UTC in milliseconds
	got := parseTimestamp("1704067200000")
	want := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestParseTimestamp_Invalid(t *testing.T) {
	got := parseTimestamp("invalid")
	if !got.IsZero() {
		t.Errorf("expected zero time for invalid timestamp, got %v", got)
	}
}
