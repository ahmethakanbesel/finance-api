package job

import "context"

type Repository interface {
	Create(ctx context.Context, j *Job) error
	Update(ctx context.Context, j *Job) error
	Get(ctx context.Context, id int64) (*Job, error)
	List(ctx context.Context, source, symbol string) ([]Job, error)
	FindActive(ctx context.Context, source, symbol string, from, to string) (*Job, error)
	ClaimPending(ctx context.Context) (*Job, error)
	RecoverStale(ctx context.Context) (int64, error)
}
