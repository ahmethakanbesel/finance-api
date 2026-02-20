package test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ahmethakanbesel/finance-api/internal/job"
	"github.com/ahmethakanbesel/finance-api/internal/platform/sqlite"
	"github.com/ahmethakanbesel/finance-api/internal/price"
	"github.com/ahmethakanbesel/finance-api/internal/rate"
	jobrepo "github.com/ahmethakanbesel/finance-api/internal/repository/job"
	pricerepo "github.com/ahmethakanbesel/finance-api/internal/repository/price"
	raterepo "github.com/ahmethakanbesel/finance-api/internal/repository/rate"
	"github.com/ahmethakanbesel/finance-api/internal/scraper"
	"github.com/ahmethakanbesel/finance-api/internal/scraper/isyatirim"
	"github.com/ahmethakanbesel/finance-api/internal/scraper/tefas"
	"github.com/ahmethakanbesel/finance-api/internal/scraper/yahoo"
	"github.com/ahmethakanbesel/finance-api/internal/server"
)

func setupE2E(t *testing.T, tefasURL, isyatirimURL string) *httptest.Server {
	t.Helper()

	db, err := sqlite.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	priceRepo := pricerepo.NewRepository(db.DB)
	jobRepo := jobrepo.NewRepository(db.DB)
	rateRepo := raterepo.NewRepository(db.DB)

	registry := scraper.NewRegistry()
	registry.Register(tefas.New(
		tefas.WithWorkers(1),
		tefas.WithHistoryEndpoint(tefasURL),
		tefas.WithBaseURL(tefasURL),
		tefas.WithReferer(tefasURL),
	))
	registry.Register(yahoo.New(
		yahoo.WithWorkers(1),
	))

	if isyatirimURL != "" {
		registry.Register(isyatirim.New(
			isyatirim.WithWorkers(1),
			isyatirim.WithEndpoint(isyatirimURL),
		))
	} else {
		registry.Register(isyatirim.New(
			isyatirim.WithWorkers(1),
		))
	}

	rateSvc := rate.NewService(rateRepo)
	jobSvc := job.NewService(jobRepo)
	priceSvc := price.NewService(priceRepo, jobRepo, registry, rateSvc)

	// Start worker pool for background job processing
	poolCtx, poolCancel := context.WithCancel(context.Background())
	pool := job.NewWorkerPool(jobRepo, priceSvc, 2)
	priceSvc.SetNotify(pool.Notify)
	poolDone := make(chan struct{})
	go func() {
		pool.Run(poolCtx)
		close(poolDone)
	}()
	// Cleanup runs LIFO: cancel pool → wait for drain → then db.Close (registered earlier)
	t.Cleanup(func() {
		poolCancel()
		<-poolDone
	})

	return httptest.NewServer(server.NewHandler(priceSvc, jobSvc))
}

// waitForJob polls the job endpoint until the job reaches a terminal status.
func waitForJob(t *testing.T, baseURL string, jobID int64) *job.Job {
	t.Helper()

	deadline := time.After(5 * time.Second)
	for {
		select {
		case <-deadline:
			t.Fatalf("timed out waiting for job %d to complete", jobID)
		default:
		}

		resp, err := http.Get(fmt.Sprintf("%s/api/v1/jobs/%d", baseURL, jobID)) //nolint:gosec // test URL
		if err != nil {
			t.Fatalf("request: %v", err)
		}

		var result struct {
			Data job.Job `json:"data"`
		}
		err = json.NewDecoder(resp.Body).Decode(&result)
		_ = resp.Body.Close()
		if err != nil {
			t.Fatalf("decode: %v", err)
		}

		if result.Data.Status == job.StatusCompleted || result.Data.Status == job.StatusFailed {
			return &result.Data
		}

		time.Sleep(50 * time.Millisecond)
	}
}

func TestE2E_Health(t *testing.T) {
	ts := setupE2E(t, "", "")
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/health") //nolint:gosec // test URL
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestE2E_ListSources(t *testing.T) {
	ts := setupE2E(t, "", "")
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/v1/sources") //nolint:gosec // test URL
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestE2E_GetPrices_TEFAS(t *testing.T) {
	// Mock TEFAS server
	mockTefas := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"recordsTotal": 2,
			"data": []map[string]any{
				{"TARIH": "1704067200000", "FONKODU": "YAC", "FIYAT": 1.23},
				{"TARIH": "1704153600000", "FONKODU": "YAC", "FIYAT": 1.24},
			},
		})
	}))
	defer mockTefas.Close()

	ts := setupE2E(t, mockTefas.URL, "")
	defer ts.Close()

	url := fmt.Sprintf("%s/api/v1/prices/YAC?source=tefas&startDate=2024-01-01&endDate=2024-01-31&currency=TRY", ts.URL)

	// First request: returns pending job
	resp, err := http.Get(url) //nolint:gosec // test URL
	if err != nil {
		t.Fatalf("request: %v", err)
	}

	var result struct {
		Message string `json:"message"`
		Data    struct {
			Prices []price.PricePoint `json:"prices"`
			Job    *job.Job           `json:"job"`
		} `json:"data"`
	}
	err = json.NewDecoder(resp.Body).Decode(&result)
	_ = resp.Body.Close()
	if err != nil {
		t.Fatal(err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	if result.Message != "ok" {
		t.Errorf("expected message 'ok', got '%s'", result.Message)
	}
	if result.Data.Job == nil {
		t.Fatal("expected job in first request")
	}

	// Wait for job to complete
	completedJob := waitForJob(t, ts.URL, result.Data.Job.ID)
	if completedJob.Status != job.StatusCompleted {
		t.Errorf("expected completed, got %s (error: %s)", completedJob.Status, completedJob.Error)
	}

	// Second request: returns cached prices
	resp2, err := http.Get(url) //nolint:gosec // test URL
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer func() { _ = resp2.Body.Close() }()

	var result2 struct {
		Message string `json:"message"`
		Data    struct {
			Prices []price.PricePoint `json:"prices"`
			Job    *job.Job           `json:"job"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp2.Body).Decode(&result2); err != nil {
		t.Fatal(err)
	}

	if len(result2.Data.Prices) != 2 {
		t.Errorf("expected 2 prices, got %d", len(result2.Data.Prices))
	}
	// Same currency — rate should be 1.0
	for _, pp := range result2.Data.Prices {
		if pp.Rate != 1.0 {
			t.Errorf("expected rate 1.0 for same currency, got %f", pp.Rate)
		}
		if pp.NativeCurrency != price.CurrencyTRY {
			t.Errorf("expected native currency TRY, got %s", pp.NativeCurrency)
		}
	}
}

func TestE2E_GetPrices_Dedup(t *testing.T) {
	callCount := 0
	mockTefas := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		// Return enough prices to cover the range (5 weekdays in Jan 1-5)
		data := make([]map[string]any, 0)
		start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		for i := 0; i < 5; i++ {
			d := start.AddDate(0, 0, i)
			ms := d.UnixMilli()
			data = append(data, map[string]any{
				"TARIH":   fmt.Sprintf("%d", ms),
				"FONKODU": "YAC",
				"FIYAT":   1.23 + float64(i)*0.01,
			})
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"recordsTotal": len(data),
			"data":         data,
		})
	}))
	defer mockTefas.Close()

	ts := setupE2E(t, mockTefas.URL, "")
	defer ts.Close()

	url := fmt.Sprintf("%s/api/v1/prices/YAC?source=tefas&startDate=2024-01-01&endDate=2024-01-05&currency=TRY", ts.URL)

	// First request - should create a pending job
	resp, err := http.Get(url) //nolint:gosec // test URL
	if err != nil {
		t.Fatalf("request: %v", err)
	}

	var result struct {
		Data struct {
			Job *job.Job `json:"job"`
		} `json:"data"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&result)
	_ = resp.Body.Close()

	if result.Data.Job == nil {
		t.Fatal("expected job in first request")
	}

	// Wait for the job to complete
	waitForJob(t, ts.URL, result.Data.Job.ID)

	// Second request - should use cache (no new scraper call)
	resp2, _ := http.Get(url) //nolint:gosec // test URL
	_ = resp2.Body.Close()

	if callCount != 1 {
		t.Errorf("expected scraper called once (dedup), got %d", callCount)
	}
}

func TestE2E_GetPrices_CSV(t *testing.T) {
	mockTefas := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"recordsTotal": 1,
			"data": []map[string]any{
				{"TARIH": "1704067200000", "FONKODU": "YAC", "FIYAT": 1.23},
			},
		})
	}))
	defer mockTefas.Close()

	ts := setupE2E(t, mockTefas.URL, "")
	defer ts.Close()

	// First request to trigger scraping
	firstURL := fmt.Sprintf("%s/api/v1/prices/YAC?source=tefas&startDate=2024-01-01&endDate=2024-01-31&currency=TRY", ts.URL)
	resp, _ := http.Get(firstURL) //nolint:gosec // test URL
	var result struct {
		Data struct {
			Job *job.Job `json:"job"`
		} `json:"data"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&result)
	_ = resp.Body.Close()

	if result.Data.Job != nil {
		waitForJob(t, ts.URL, result.Data.Job.ID)
	}

	// Now request CSV format
	csvURL := fmt.Sprintf("%s/api/v1/prices/YAC?source=tefas&startDate=2024-01-01&endDate=2024-01-31&currency=TRY&format=csv", ts.URL)
	resp, err := http.Get(csvURL) //nolint:gosec // test URL
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.Header.Get("Content-Type") != "text/csv" {
		t.Errorf("expected text/csv, got %s", resp.Header.Get("Content-Type"))
	}
}

func TestE2E_GetPrices_InvalidParams(t *testing.T) {
	ts := setupE2E(t, "", "")
	defer ts.Close()

	// Missing source
	resp, _ := http.Get(ts.URL + "/api/v1/prices/YAC?startDate=2024-01-01&currency=TRY") //nolint:gosec // test URL
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400 for missing source, got %d", resp.StatusCode)
	}

	// Missing startDate
	resp, _ = http.Get(ts.URL + "/api/v1/prices/YAC?source=tefas&currency=TRY") //nolint:gosec // test URL
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400 for missing startDate, got %d", resp.StatusCode)
	}
}

func TestE2E_Jobs(t *testing.T) {
	mockTefas := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"recordsTotal": 1,
			"data": []map[string]any{
				{"TARIH": "1704067200000", "FONKODU": "YAC", "FIYAT": 1.23},
			},
		})
	}))
	defer mockTefas.Close()

	ts := setupE2E(t, mockTefas.URL, "")
	defer ts.Close()

	// Trigger a scrape to create a job
	url := fmt.Sprintf("%s/api/v1/prices/YAC?source=tefas&startDate=2024-01-01&endDate=2024-01-31&currency=TRY", ts.URL)
	resp, _ := http.Get(url) //nolint:gosec // test URL
	var firstResult struct {
		Data struct {
			Job *job.Job `json:"job"`
		} `json:"data"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&firstResult)
	_ = resp.Body.Close()

	if firstResult.Data.Job != nil {
		waitForJob(t, ts.URL, firstResult.Data.Job.ID)
	}

	// List jobs
	resp, err := http.Get(ts.URL + "/api/v1/jobs") //nolint:gosec // test URL
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	var result struct {
		Data []job.Job `json:"data"`
	}
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Data) == 0 {
		t.Error("expected at least 1 job")
	}

	// Get specific job
	resp2, err := http.Get(fmt.Sprintf("%s/api/v1/jobs/%d", ts.URL, result.Data[0].ID)) //nolint:gosec // test URL
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer func() { _ = resp2.Body.Close() }()
	if resp2.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp2.StatusCode)
	}
}

func TestE2E_GetPrices_Isyatirim(t *testing.T) {
	mockIsyatirim := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("endeks") != "ALTINS1" {
			t.Errorf("expected endeks=ALTINS1, got %s", q.Get("endeks"))
		}

		resp := map[string]any{
			"data": [][]any{
				{1735772400000.0, 36.65},
				{1735858800000.0, 36.74},
				{1735945200000.0, 36.80},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer mockIsyatirim.Close()

	ts := setupE2E(t, "", mockIsyatirim.URL)
	defer ts.Close()

	url := fmt.Sprintf("%s/api/v1/prices/ALTINS1?source=isyatirim&startDate=2025-01-01&endDate=2025-01-31&currency=TRY", ts.URL)

	// First request: creates pending job
	resp, err := http.Get(url) //nolint:gosec // test URL
	if err != nil {
		t.Fatalf("request: %v", err)
	}

	var result struct {
		Message string `json:"message"`
		Data    struct {
			Prices []price.PricePoint `json:"prices"`
			Job    *job.Job           `json:"job"`
		} `json:"data"`
	}
	err = json.NewDecoder(resp.Body).Decode(&result)
	_ = resp.Body.Close()
	if err != nil {
		t.Fatal(err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	if result.Message != "ok" {
		t.Errorf("expected message 'ok', got '%s'", result.Message)
	}
	if result.Data.Job == nil {
		t.Fatal("expected job in first request")
	}

	// Wait for job to complete
	completedJob := waitForJob(t, ts.URL, result.Data.Job.ID)
	if completedJob.Status != job.StatusCompleted {
		t.Errorf("expected completed, got %s (error: %s)", completedJob.Status, completedJob.Error)
	}

	// Second request: prices are now cached
	resp2, err := http.Get(url) //nolint:gosec // test URL
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer func() { _ = resp2.Body.Close() }()

	var result2 struct {
		Message string `json:"message"`
		Data    struct {
			Prices []price.PricePoint `json:"prices"`
			Job    *job.Job           `json:"job"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp2.Body).Decode(&result2); err != nil {
		t.Fatal(err)
	}

	if len(result2.Data.Prices) != 3 {
		t.Errorf("expected 3 prices, got %d", len(result2.Data.Prices))
	}
	for _, pp := range result2.Data.Prices {
		if pp.Source != price.SourceIsyatirim {
			t.Errorf("expected source isyatirim, got %s", pp.Source)
		}
		if pp.NativeCurrency != price.CurrencyTRY {
			t.Errorf("expected native currency TRY, got %s", pp.NativeCurrency)
		}
	}
}
