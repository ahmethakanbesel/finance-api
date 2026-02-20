package price

import (
	"context"
	"time"
)

type Repository interface {
	SavePrices(ctx context.Context, prices []Price) (int64, error)
	ListPrices(ctx context.Context, source Source, symbol string, from, to time.Time) ([]Price, error)
	ExistingDates(ctx context.Context, source Source, symbol string, from, to time.Time) (map[time.Time]bool, error)
}
