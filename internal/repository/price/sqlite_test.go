package price

import (
	"context"
	"testing"
	"time"

	"github.com/ahmethakanbesel/finance-api/internal/platform/sqlite"
	domain "github.com/ahmethakanbesel/finance-api/internal/price"
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

func TestSavePrices_And_ListPrices(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db.DB)
	ctx := context.Background()

	prices := []domain.Price{
		{Source: domain.SourceTefas, Symbol: "YAC", Date: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), ClosePrice: 1.23, Currency: domain.CurrencyTRY},
		{Source: domain.SourceTefas, Symbol: "YAC", Date: time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC), ClosePrice: 1.24, Currency: domain.CurrencyTRY},
		{Source: domain.SourceTefas, Symbol: "YAC", Date: time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC), ClosePrice: 1.25, Currency: domain.CurrencyTRY},
	}

	n, err := repo.SavePrices(ctx, prices)
	if err != nil {
		t.Fatalf("save prices: %v", err)
	}
	if n != 3 {
		t.Errorf("expected 3 rows inserted, got %d", n)
	}

	// List them back
	got, err := repo.ListPrices(ctx, domain.SourceTefas, "YAC",
		time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("list prices: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("expected 3 prices, got %d", len(got))
	}
	if got[0].ClosePrice != 1.23 {
		t.Errorf("expected 1.23, got %f", got[0].ClosePrice)
	}
}

func TestSavePrices_Idempotent(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db.DB)
	ctx := context.Background()

	prices := []domain.Price{
		{Source: domain.SourceTefas, Symbol: "YAC", Date: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), ClosePrice: 1.23, Currency: domain.CurrencyTRY},
	}

	n1, err := repo.SavePrices(ctx, prices)
	if err != nil {
		t.Fatalf("first save: %v", err)
	}
	if n1 != 1 {
		t.Errorf("expected 1 row, got %d", n1)
	}

	// Same data again -- should be ignored
	n2, err := repo.SavePrices(ctx, prices)
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

	prices := []domain.Price{
		{Source: domain.SourceTefas, Symbol: "YAC", Date: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), ClosePrice: 1.23, Currency: domain.CurrencyTRY},
		{Source: domain.SourceTefas, Symbol: "YAC", Date: time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC), ClosePrice: 1.25, Currency: domain.CurrencyTRY},
	}
	if _, err := repo.SavePrices(ctx, prices); err != nil {
		t.Fatal(err)
	}

	dates, err := repo.ExistingDates(ctx, domain.SourceTefas, "YAC",
		time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2024, 1, 5, 0, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("existing dates: %v", err)
	}
	if len(dates) != 2 {
		t.Fatalf("expected 2 dates, got %d", len(dates))
	}
	if !dates[time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)] {
		t.Error("expected 2024-01-01 to exist")
	}
	if dates[time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)] {
		t.Error("expected 2024-01-02 to not exist")
	}
}

func TestSavePrices_Empty(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db.DB)
	n, err := repo.SavePrices(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 0 {
		t.Errorf("expected 0, got %d", n)
	}
}
