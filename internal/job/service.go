package job

import (
	"context"
	"log/slog"
)

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) RecoverStaleJobs(ctx context.Context) error {
	n, err := s.repo.RecoverStale(ctx)
	if err != nil {
		return err
	}
	if n > 0 {
		slog.Info("re-queued interrupted jobs", "count", n)
	}
	return nil
}

func (s *Service) Get(ctx context.Context, req GetJobRequest) (*Job, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	return s.repo.Get(ctx, req.ID)
}

func (s *Service) List(ctx context.Context, req ListJobsRequest) ([]Job, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	return s.repo.List(ctx, req.Source, req.Symbol)
}
