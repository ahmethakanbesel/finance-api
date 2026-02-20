package job

import (
	"context"
	"testing"
	"time"

	domain "github.com/ahmethakanbesel/finance-api/internal/job"
	"github.com/ahmethakanbesel/finance-api/internal/platform/sqlite"
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

func TestCreate_And_Get(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db.DB)
	ctx := context.Background()

	j := &domain.Job{
		Source:    "tefas",
		Symbol:    "YAC",
		StartDate: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC),
		Status:    domain.StatusPending,
	}

	if err := repo.Create(ctx, j); err != nil {
		t.Fatalf("create: %v", err)
	}
	if j.ID == 0 {
		t.Fatal("expected non-zero ID")
	}

	got, err := repo.Get(ctx, j.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Symbol != "YAC" {
		t.Errorf("expected YAC, got %s", got.Symbol)
	}
	if got.Status != domain.StatusPending {
		t.Errorf("expected pending, got %s", got.Status)
	}
}

func TestUpdate(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db.DB)
	ctx := context.Background()

	j := &domain.Job{
		Source:    "tefas",
		Symbol:    "YAC",
		StartDate: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC),
		Status:    domain.StatusPending,
	}
	if err := repo.Create(ctx, j); err != nil {
		t.Fatal(err)
	}

	j.Status = domain.StatusCompleted
	j.RecordsCount = 20
	if err := repo.Update(ctx, j); err != nil {
		t.Fatalf("update: %v", err)
	}

	got, _ := repo.Get(ctx, j.ID)
	if got.Status != domain.StatusCompleted {
		t.Errorf("expected completed, got %s", got.Status)
	}
	if got.RecordsCount != 20 {
		t.Errorf("expected 20, got %d", got.RecordsCount)
	}
}

func TestList(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db.DB)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		if err := repo.Create(ctx, &domain.Job{
			Source:    "tefas",
			Symbol:    "YAC",
			StartDate: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			EndDate:   time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC),
			Status:    domain.StatusPending,
		}); err != nil {
			t.Fatal(err)
		}
	}

	jobs, err := repo.List(ctx, "tefas", "")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(jobs) != 3 {
		t.Errorf("expected 3, got %d", len(jobs))
	}

	jobs, err = repo.List(ctx, "yahoo", "")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(jobs) != 0 {
		t.Errorf("expected 0, got %d", len(jobs))
	}
}

func TestRecoverStale(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db.DB)
	ctx := context.Background()

	if err := repo.Create(ctx, &domain.Job{
		Source: "tefas", Symbol: "YAC",
		StartDate: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC),
		Status:    domain.StatusRunning,
	}); err != nil {
		t.Fatal(err)
	}
	if err := repo.Create(ctx, &domain.Job{
		Source: "tefas", Symbol: "YAC",
		StartDate: time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2024, 2, 28, 0, 0, 0, 0, time.UTC),
		Status:    domain.StatusPending,
	}); err != nil {
		t.Fatal(err)
	}
	if err := repo.Create(ctx, &domain.Job{
		Source: "tefas", Symbol: "YAC",
		StartDate: time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2024, 3, 31, 0, 0, 0, 0, time.UTC),
		Status:    domain.StatusCompleted,
	}); err != nil {
		t.Fatal(err)
	}

	n, err := repo.RecoverStale(ctx)
	if err != nil {
		t.Fatalf("recover: %v", err)
	}
	if n != 1 {
		t.Errorf("expected 1 recovered (running→pending), got %d", n)
	}

	// The recovered job should now be pending.
	j, err := repo.Get(ctx, 1) // the running job
	if err != nil {
		t.Fatal(err)
	}
	if j.Status != domain.StatusPending {
		t.Errorf("expected status pending, got %s", j.Status)
	}

	// Running it again should be a no-op (no more running jobs).
	n2, err := repo.RecoverStale(ctx)
	if err != nil {
		t.Fatalf("recover again: %v", err)
	}
	if n2 != 0 {
		t.Errorf("expected 0, got %d", n2)
	}
}

func TestFindActive(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db.DB)
	ctx := context.Background()

	if err := repo.Create(ctx, &domain.Job{
		Source: "tefas", Symbol: "YAC",
		StartDate: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC),
		Status:    domain.StatusRunning,
	}); err != nil {
		t.Fatal(err)
	}

	got, err := repo.FindActive(ctx, "tefas", "YAC", "2024-01-01", "2024-01-31")
	if err != nil {
		t.Fatalf("find active: %v", err)
	}
	if got == nil {
		t.Fatal("expected active job")
	}

	// No match
	got, err = repo.FindActive(ctx, "yahoo", "AAPL", "2024-01-01", "2024-01-31")
	if err != nil {
		t.Fatalf("find active: %v", err)
	}
	if got != nil {
		t.Error("expected nil for non-matching query")
	}
}

func TestGet_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db.DB)
	_, err := repo.Get(context.Background(), 999)
	if err == nil {
		t.Fatal("expected error for missing job")
	}
}
