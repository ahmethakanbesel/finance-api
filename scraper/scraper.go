package scraper

import (
	"context"
	"time"
)

const (
	DateFormat = "2006-01-02"
)

type Scraper interface {
	GetSymbolData(ctx context.Context, symbol string, startDate, endDate time.Time) (<-chan *SymbolPrice, error)
}
