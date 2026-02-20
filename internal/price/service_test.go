package price

import (
	"context"
	"testing"
	"time"

	"github.com/ahmethakanbesel/finance-api/internal/job"
	"github.com/ahmethakanbesel/finance-api/internal/rate"
	"github.com/ahmethakanbesel/finance-api/internal/scraper"
)

// --- mock price repo ---
type mockPriceRepo struct {
	prices []Price
	dates  map[time.Time]bool
}

func (m *mockPriceRepo) SavePrices(_ context.Context, prices []Price) (int64, error) {
	m.prices = append(m.prices, prices...)
	return int64(len(prices)), nil
}

func (m *mockPriceRepo) ListPrices(_ context.Context, _ Source, _ string, _, _ time.Time) ([]Price, error) {
	return m.prices, nil
}

func (m *mockPriceRepo) ExistingDates(_ context.Context, _ Source, _ string, _, _ time.Time) (map[time.Time]bool, error) {
	if m.dates == nil {
		return make(map[time.Time]bool), nil
	}
	return m.dates, nil
}

// --- mock job repo ---
type mockJobRepo struct {
	jobs   []*job.Job
	nextID int64
}

func (m *mockJobRepo) Create(_ context.Context, j *job.Job) error {
	m.nextID++
	j.ID = m.nextID
	cp := *j
	m.jobs = append(m.jobs, &cp)
	return nil
}
func (m *mockJobRepo) Update(_ context.Context, j *job.Job) error {
	for i, existing := range m.jobs {
		if existing.ID == j.ID {
			cp := *j
			m.jobs[i] = &cp
			return nil
		}
	}
	return nil
}
func (m *mockJobRepo) Get(_ context.Context, id int64) (*job.Job, error) {
	for _, j := range m.jobs {
		if j.ID == id {
			return j, nil
		}
	}
	return nil, nil
}
func (m *mockJobRepo) List(_ context.Context, _, _ string) ([]job.Job, error) { return nil, nil }
func (m *mockJobRepo) FindActive(_ context.Context, _, _, _, _ string) (*job.Job, error) {
	return nil, nil
}
func (m *mockJobRepo) ClaimPending(_ context.Context) (*job.Job, error) {
	return nil, nil
}
func (m *mockJobRepo) RecoverStale(_ context.Context) (int64, error) { return 0, nil }

// --- mock scraper ---
type mockScraper struct {
	prices         []scraper.ScrapedPrice
	nativeCurrency string
}

func (m *mockScraper) Source() string { return "tefas" }

func (m *mockScraper) NativeCurrency(_ string) string {
	if m.nativeCurrency != "" {
		return m.nativeCurrency
	}
	return "TRY"
}

func (m *mockScraper) Scrape(_ context.Context, _ string, _, _ time.Time) ([]scraper.ScrapedPrice, error) {
	return m.prices, nil
}

// --- mock rate repo ---
type mockRateRepo struct {
	rates []rate.Rate
	dates map[time.Time]bool
}

func (m *mockRateRepo) SaveRates(_ context.Context, rates []rate.Rate) (int64, error) {
	m.rates = append(m.rates, rates...)
	return int64(len(rates)), nil
}

func (m *mockRateRepo) ListRates(_ context.Context, _ string, _, _ time.Time) ([]rate.Rate, error) {
	return m.rates, nil
}

func (m *mockRateRepo) ExistingDates(_ context.Context, _ string, _, _ time.Time) (map[time.Time]bool, error) {
	if m.dates == nil {
		return make(map[time.Time]bool), nil
	}
	return m.dates, nil
}

func TestGetPrices_QueueJob(t *testing.T) {
	priceRepo := &mockPriceRepo{}
	jobRepo := &mockJobRepo{}
	ms := &mockScraper{
		prices: []scraper.ScrapedPrice{
			{Date: time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC), ClosePrice: 1.23},
			{Date: time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC), ClosePrice: 1.24},
		},
	}

	reg := scraper.NewRegistry()
	reg.Register(ms)

	notified := false
	svc := NewService(priceRepo, jobRepo, reg, nil)
	svc.SetNotify(func() { notified = true })

	resp, err := svc.GetPrices(context.Background(), GetPricesRequest{
		Source:    SourceTefas,
		Symbol:    "YAC",
		Currency:  CurrencyTRY,
		StartDate: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2024, 1, 5, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Job == nil {
		t.Fatal("expected job to be created")
	}
	if resp.Job.Status != job.StatusPending {
		t.Errorf("expected pending, got %s", resp.Job.Status)
	}
	if !notified {
		t.Error("expected notify to be called")
	}
	// GetPrices should not scrape — prices should be empty (nothing in DB)
	if len(resp.Prices) != 0 {
		t.Errorf("expected 0 prices (async), got %d", len(resp.Prices))
	}
}

func TestProcess_ScrapeAndSave(t *testing.T) {
	priceRepo := &mockPriceRepo{}
	jobRepo := &mockJobRepo{}
	ms := &mockScraper{
		prices: []scraper.ScrapedPrice{
			{Date: time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC), ClosePrice: 1.23},
			{Date: time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC), ClosePrice: 1.24},
		},
	}

	reg := scraper.NewRegistry()
	reg.Register(ms)

	svc := NewService(priceRepo, jobRepo, reg, nil)

	j := &job.Job{
		ID:        1,
		Source:    "tefas",
		Symbol:    "YAC",
		StartDate: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2024, 1, 5, 0, 0, 0, 0, time.UTC),
		Status:    job.StatusRunning,
	}
	// Add to repo so Update can find it
	jobRepo.nextID = 0
	_ = jobRepo.Create(context.Background(), j)

	if err := svc.Process(context.Background(), j); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if j.Status != job.StatusCompleted {
		t.Errorf("expected completed, got %s", j.Status)
	}
	if len(priceRepo.prices) != 2 {
		t.Errorf("expected 2 saved prices, got %d", len(priceRepo.prices))
	}
}

func TestGetPrices_ServedFromCache(t *testing.T) {
	// Pre-fill enough dates to hit >80% coverage
	dates := make(map[time.Time]bool)
	prices := make([]Price, 0)
	for d := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC); d.Before(time.Date(2024, 1, 6, 0, 0, 0, 0, time.UTC)); d = d.AddDate(0, 0, 1) {
		dates[d] = true
		prices = append(prices, Price{Source: SourceTefas, Symbol: "YAC", Date: d, ClosePrice: 1.0, Currency: CurrencyTRY})
	}

	priceRepo := &mockPriceRepo{dates: dates, prices: prices}
	jobRepo := &mockJobRepo{}
	reg := scraper.NewRegistry()
	reg.Register(&mockScraper{})

	svc := NewService(priceRepo, jobRepo, reg, nil)

	resp, err := svc.GetPrices(context.Background(), GetPricesRequest{
		Source:    SourceTefas,
		Symbol:    "YAC",
		Currency:  CurrencyTRY,
		StartDate: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2024, 1, 5, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Job != nil {
		t.Error("expected no job (served from cache)")
	}
}

func TestGetPrices_ValidationError(t *testing.T) {
	svc := NewService(nil, nil, nil, nil)

	_, err := svc.GetPrices(context.Background(), GetPricesRequest{
		Symbol: "A", // too short
	})
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestGetPrices_ConvertTRYtoUSD(t *testing.T) {
	priceRepo := &mockPriceRepo{}
	jobRepo := &mockJobRepo{}
	ms := &mockScraper{
		nativeCurrency: "TRY",
		prices: []scraper.ScrapedPrice{
			{Date: time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC), ClosePrice: 30.0},
			{Date: time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC), ClosePrice: 60.0},
		},
	}

	reg := scraper.NewRegistry()
	reg.Register(ms)

	// Set up rate service with pre-filled rates
	rateRepo := &mockRateRepo{
		rates: []rate.Rate{
			{Pair: rate.PairUSDTRY, Date: time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC), Rate: 30.0},
			{Pair: rate.PairUSDTRY, Date: time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC), Rate: 30.0},
		},
		dates: map[time.Time]bool{
			time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC): true,
			time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC): true,
		},
	}
	rateSvc := rate.NewService(rateRepo)

	svc := NewService(priceRepo, jobRepo, reg, rateSvc)

	// First call: queues a pending job (no prices in DB yet)
	_, _ = svc.GetPrices(context.Background(), GetPricesRequest{
		Source:    SourceTefas,
		Symbol:    "YAC",
		Currency:  CurrencyUSD,
		StartDate: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2024, 1, 5, 0, 0, 0, 0, time.UTC),
	})

	// Simulate worker processing the job
	j := &job.Job{
		ID:        jobRepo.jobs[0].ID,
		Source:    "tefas",
		Symbol:    "YAC",
		StartDate: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2024, 1, 5, 0, 0, 0, 0, time.UTC),
		Status:    job.StatusRunning,
	}
	if err := svc.Process(context.Background(), j); err != nil {
		t.Fatalf("process error: %v", err)
	}

	// Now prices are in DB — second call should return them with conversion
	resp, err := svc.GetPrices(context.Background(), GetPricesRequest{
		Source:    SourceTefas,
		Symbol:    "YAC",
		Currency:  CurrencyUSD,
		StartDate: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2024, 1, 5, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resp.Prices) != 2 {
		t.Fatalf("expected 2 prices, got %d", len(resp.Prices))
	}

	// 30 TRY / 30 USDTRY = 1.0 USD
	if resp.Prices[0].ClosePrice != 1.0 {
		t.Errorf("expected converted price 1.0, got %f", resp.Prices[0].ClosePrice)
	}
	if resp.Prices[0].NativePrice != 30.0 {
		t.Errorf("expected native price 30.0, got %f", resp.Prices[0].NativePrice)
	}
	if resp.Prices[0].Rate != 30.0 {
		t.Errorf("expected rate 30.0, got %f", resp.Prices[0].Rate)
	}
	if resp.Prices[0].NativeCurrency != CurrencyTRY {
		t.Errorf("expected native currency TRY, got %s", resp.Prices[0].NativeCurrency)
	}
	if resp.Prices[0].Currency != CurrencyUSD {
		t.Errorf("expected currency USD, got %s", resp.Prices[0].Currency)
	}

	// 60 TRY / 30 USDTRY = 2.0 USD
	if resp.Prices[1].ClosePrice != 2.0 {
		t.Errorf("expected converted price 2.0, got %f", resp.Prices[1].ClosePrice)
	}
}

func TestGetPrices_ConvertUSDtoTRY(t *testing.T) {
	priceRepo := &mockPriceRepo{}
	jobRepo := &mockJobRepo{}

	// Yahoo scraper with USD native currency
	ms := &mockScraper{
		nativeCurrency: "USD",
		prices: []scraper.ScrapedPrice{
			{Date: time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC), ClosePrice: 100.0},
		},
	}
	reg := scraper.NewRegistry()
	reg.Register(ms)

	rateRepo := &mockRateRepo{
		rates: []rate.Rate{
			{Pair: rate.PairUSDTRY, Date: time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC), Rate: 30.0},
		},
		dates: map[time.Time]bool{
			time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC): true,
		},
	}
	rateSvc := rate.NewService(rateRepo)

	svc := NewService(priceRepo, jobRepo, reg, rateSvc)

	// First call: queues a pending job
	_, _ = svc.GetPrices(context.Background(), GetPricesRequest{
		Source:    SourceTefas,
		Symbol:    "AAPL",
		Currency:  CurrencyTRY,
		StartDate: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2024, 1, 5, 0, 0, 0, 0, time.UTC),
	})

	// Simulate worker processing
	j := &job.Job{
		ID:        jobRepo.jobs[0].ID,
		Source:    "tefas",
		Symbol:    "AAPL",
		StartDate: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2024, 1, 5, 0, 0, 0, 0, time.UTC),
		Status:    job.StatusRunning,
	}
	if err := svc.Process(context.Background(), j); err != nil {
		t.Fatalf("process error: %v", err)
	}

	// Second call: prices are in DB
	resp, err := svc.GetPrices(context.Background(), GetPricesRequest{
		Source:    SourceTefas,
		Symbol:    "AAPL",
		Currency:  CurrencyTRY,
		StartDate: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2024, 1, 5, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resp.Prices) != 1 {
		t.Fatalf("expected 1 price, got %d", len(resp.Prices))
	}

	// 100 USD * 30 USDTRY = 3000 TRY
	if resp.Prices[0].ClosePrice != 3000.0 {
		t.Errorf("expected converted price 3000.0, got %f", resp.Prices[0].ClosePrice)
	}
	if resp.Prices[0].NativePrice != 100.0 {
		t.Errorf("expected native price 100.0, got %f", resp.Prices[0].NativePrice)
	}
	if resp.Prices[0].Rate != 30.0 {
		t.Errorf("expected rate 30.0, got %f", resp.Prices[0].Rate)
	}
}
