package scraper

import (
	"time"
)

const (
	DateFormat = "2006-01-02"
)

type Scraper interface {
	GetSymbolData(symbol string, startDate, endDate time.Time) (<-chan *SymbolPrice, error)
}
