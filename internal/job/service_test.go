package job

import (
	"context"
	"sync"
	"testing"
)

type mockRepo struct {
	mu         sync.Mutex
	jobs       map[int64]*Job
	nextID     int64
	staleCount int64
	recoverErr error
}

func newMockRepo() *mockRepo {
	return &mockRepo{jobs: make(map[int64]*Job), nextID: 1}
}

func (m *mockRepo) Create(_ context.Context, j *Job) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	j.ID = m.nextID
	m.nextID++
	cp := *j
	m.jobs[j.ID] = &cp
	return nil
}

func (m *mockRepo) Update(_ context.Context, j *Job) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := *j
	m.jobs[j.ID] = &cp
	return nil
}

func (m *mockRepo) Get(_ context.Context, id int64) (*Job, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	j, ok := m.jobs[id]
	if !ok {
		return nil, &notFoundErr{}
	}
	cp := *j
	return &cp, nil
}

func (m *mockRepo) List(_ context.Context, source, symbol string) ([]Job, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]Job, 0, len(m.jobs))
	for _, j := range m.jobs {
		if source != "" && j.Source != source {
			continue
		}
		if symbol != "" && j.Symbol != symbol {
			continue
		}
		result = append(result, *j)
	}
	return result, nil
}

func (m *mockRepo) FindActive(_ context.Context, _, _, _, _ string) (*Job, error) {
	return nil, nil
}

func (m *mockRepo) ClaimPending(_ context.Context) (*Job, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, j := range m.jobs {
		if j.Status == StatusPending {
			j.Status = StatusRunning
			cp := *j
			return &cp, nil
		}
	}
	return nil, nil
}

func (m *mockRepo) RecoverStale(_ context.Context) (int64, error) {
	return m.staleCount, m.recoverErr
}

type notFoundErr struct{}

func (e *notFoundErr) Error() string { return "not found" }

func TestService_RecoverStaleJobs(t *testing.T) {
	repo := newMockRepo()
	repo.staleCount = 3
	svc := NewService(repo)

	if err := svc.RecoverStaleJobs(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestService_Get(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo)
	ctx := context.Background()

	if err := repo.Create(ctx, &Job{Source: "tefas", Symbol: "YAC", Status: StatusPending}); err != nil {
		t.Fatal(err)
	}

	got, err := svc.Get(ctx, GetJobRequest{ID: 1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Symbol != "YAC" {
		t.Errorf("expected YAC, got %s", got.Symbol)
	}
}

func TestService_Get_InvalidID(t *testing.T) {
	svc := NewService(newMockRepo())
	_, err := svc.Get(context.Background(), GetJobRequest{ID: 0})
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestService_List(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo)
	ctx := context.Background()

	if err := repo.Create(ctx, &Job{Source: "tefas", Symbol: "YAC", Status: StatusPending}); err != nil {
		t.Fatal(err)
	}
	if err := repo.Create(ctx, &Job{Source: "yahoo", Symbol: "AAPL", Status: StatusPending}); err != nil {
		t.Fatal(err)
	}

	jobs, err := svc.List(ctx, ListJobsRequest{Source: "tefas"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(jobs) != 1 {
		t.Errorf("expected 1 job, got %d", len(jobs))
	}
}
