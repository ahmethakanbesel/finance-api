package rate

import (
	"context"
	"time"
)

type Repository interface {
	SaveRates(ctx context.Context, rates []Rate) (int64, error)
	ListRates(ctx context.Context, pair string, from, to time.Time) ([]Rate, error)
	ExistingDates(ctx context.Context, pair string, from, to time.Time) (map[time.Time]bool, error)
}
