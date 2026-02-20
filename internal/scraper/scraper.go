package scraper

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type ScrapedPrice struct {
	Date       time.Time
	ClosePrice float64
}

type Scraper interface {
	Source() string
	NativeCurrency(symbol string) string
	Scrape(ctx context.Context, symbol string, from, to time.Time) ([]ScrapedPrice, error)
}

type Registry struct {
	mu       sync.RWMutex
	scrapers map[string]Scraper
}

func NewRegistry() *Registry {
	return &Registry{
		scrapers: make(map[string]Scraper),
	}
}

func (r *Registry) Register(s Scraper) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.scrapers[s.Source()] = s
}

func (r *Registry) Get(source string) (Scraper, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	s, ok := r.scrapers[source]
	if !ok {
		return nil, fmt.Errorf("scraper not found for source: %s", source)
	}
	return s, nil
}

func (r *Registry) Sources() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	sources := make([]string, 0, len(r.scrapers))
	for src := range r.scrapers {
		sources = append(sources, src)
	}
	return sources
}
