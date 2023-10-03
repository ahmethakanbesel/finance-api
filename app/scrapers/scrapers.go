package scrapers

import (
	"fmt"
	"math"
	"sync"
	"time"
)

const (
	dateFormat = "2006-01-02"
)

type SymbolPrice struct {
	Date  string
	Close float64
}

type Scraper interface {
	GetSymbolData(symbol string, startDate string, endDate string) (<-chan SymbolPrice, error)
	scrapeChunk(symbol string, startDate time.Time, endDate time.Time) (<-chan SymbolPrice, error)
	getChunkSize() int
}

func CreateScraper(source string) (Scraper, error) {
	switch source {
	case "yahoo":
		return &YahooScraper{}, nil
	case "tefas":
		return &TefasScraper{}, nil
	default:
		return nil, fmt.Errorf("invalid source: %s", source)
	}
}

func scrape(scraper Scraper, symbol string, startDate string, endDate string) (<-chan SymbolPrice, error) {
	startDateTime, err := time.Parse(dateFormat, startDate)
	if err != nil {
		return nil, fmt.Errorf("error parsing start date: %v", err)
	}

	endDateTime, err := time.Parse(dateFormat, endDate)
	if err != nil {
		return nil, fmt.Errorf("error parsing end date: %v", err)
	}

	chunkSize := scraper.getChunkSize()
	numChunks := int(math.Ceil(float64(endDateTime.Sub(startDateTime).Hours()) / float64(24*chunkSize)))
	resultChan := make(chan SymbolPrice, numChunks)

	var wg sync.WaitGroup
	// Split the date range into smaller chunks
	// It is needed for bypassing the rate limit of the source
	go func() {
		defer close(resultChan)
		for currentStartDate := startDateTime; currentStartDate.Before(endDateTime); currentStartDate = currentStartDate.AddDate(0, 0, chunkSize) {
			currentEndDate := currentStartDate.AddDate(0, 0, chunkSize-1)
			if currentEndDate.After(endDateTime) {
				currentEndDate = endDateTime
			}

			wg.Add(1)
			go func(symbol string, startDate, endDate time.Time) {
				chunkData, err := scraper.scrapeChunk(symbol, startDate, endDate)
				if err != nil {
					wg.Done()
					return
				}

				wg.Add(1)
				go func() {
					for data := range chunkData {
						resultChan <- data
					}
					wg.Done()
				}()
				wg.Done()
			}(symbol, currentStartDate, currentEndDate)
		}
		wg.Wait()
	}()

	return resultChan, nil
}
