package rate

import (
	"context"
	"testing"
	"time"

	"github.com/ahmethakanbesel/finance-api/internal/platform/sqlite"
	domain "github.com/ahmethakanbesel/finance-api/internal/rate"
)

func setupTestDB(t *testing.T) *sqlite.DB {
	t.Helper()
	db, err := sqlite.Open(":memory:")
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func TestSaveRates_And_ListRates(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db.DB)
	ctx := context.Background()

	rates := []domain.Rate{
		{Pair: domain.PairUSDTRY, Date: time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC), Rate: 29.50},
		{Pair: domain.PairUSDTRY, Date: time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC), Rate: 29.55},
		{Pair: domain.PairUSDTRY, Date: time.Date(2024, 1, 4, 0, 0, 0, 0, time.UTC), Rate: 29.60},
	}

	n, err := repo.SaveRates(ctx, rates)
	if err != nil {
		t.Fatalf("save rates: %v", err)
	}
	if n != 3 {
		t.Errorf("expected 3 rows inserted, got %d", n)
	}

	got, err := repo.ListRates(ctx, domain.PairUSDTRY,
		time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
		time.Date(2024, 1, 4, 0, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("list rates: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("expected 3 rates, got %d", len(got))
	}
	if got[0].Rate != 29.50 {
		t.Errorf("expected 29.50, got %f", got[0].Rate)
	}
}

func TestSaveRates_Idempotent(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db.DB)
	ctx := context.Background()

	rates := []domain.Rate{
		{Pair: domain.PairUSDTRY, Date: time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC), Rate: 29.50},
	}

	n1, err := repo.SaveRates(ctx, rates)
	if err != nil {
		t.Fatalf("first save: %v", err)
	}
	if n1 != 1 {
		t.Errorf("expected 1 row, got %d", n1)
	}

	n2, err := repo.SaveRates(ctx, rates)
	if err != nil {
		t.Fatalf("second save: %v", err)
	}
	if n2 != 0 {
		t.Errorf("expected 0 rows (idempotent), got %d", n2)
	}
}

func TestExistingDates(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db.DB)
	ctx := context.Background()

	rates := []domain.Rate{
		{Pair: domain.PairUSDTRY, Date: time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC), Rate: 29.50},
		{Pair: domain.PairUSDTRY, Date: time.Date(2024, 1, 4, 0, 0, 0, 0, time.UTC), Rate: 29.60},
	}
	if _, err := repo.SaveRates(ctx, rates); err != nil {
		t.Fatal(err)
	}

	dates, err := repo.ExistingDates(ctx, domain.PairUSDTRY,
		time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2024, 1, 5, 0, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("existing dates: %v", err)
	}
	if len(dates) != 2 {
		t.Fatalf("expected 2 dates, got %d", len(dates))
	}
	if !dates[time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)] {
		t.Error("expected 2024-01-02 to exist")
	}
	if dates[time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC)] {
		t.Error("expected 2024-01-03 to not exist")
	}
}

func TestSaveRates_Empty(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db.DB)
	n, err := repo.SaveRates(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 0 {
		t.Errorf("expected 0, got %d", n)
	}
}
