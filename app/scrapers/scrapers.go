package scrapers

import (
	"fmt"
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
	GetSymbolData(symbol string, startDate string, endDate string) ([]SymbolPrice, error)
	scrapeChunk(symbol string, startDate time.Time, endDate time.Time) ([]SymbolPrice, error)
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

func scrape(scraper Scraper, symbol string, startDate string, endDate string) ([]SymbolPrice, error) {
	// Convert start and end dates to time.Time objects for easier manipulation
	startDateTime, err := time.Parse(dateFormat, startDate)
	if err != nil {
		return nil, fmt.Errorf("error parsing start date: %v", err)
	}

	endDateTime, err := time.Parse(dateFormat, endDate)
	if err != nil {
		return nil, fmt.Errorf("error parsing end date: %v", err)
	}

	var combinedData []SymbolPrice
	chunkSize := scraper.getChunkSize()

	// Split the date range into smaller chunks
	for currentStartDate := startDateTime; currentStartDate.Before(endDateTime); currentStartDate = currentStartDate.AddDate(0, 0, chunkSize) {
		// Calculate the current end date for the chunk
		currentEndDate := currentStartDate.AddDate(0, 0, chunkSize-1)
		if currentEndDate.After(endDateTime) {
			currentEndDate = endDateTime
		}

		// Perform the scrape for the current chunk and aggregate the results
		chunkData, err := scraper.scrapeChunk(symbol, currentStartDate, currentEndDate)
		if err != nil {
			return nil, fmt.Errorf("error scraping chunk: %v", err)
		}
		combinedData = append(combinedData, chunkData...)
	}

	return combinedData, nil
}
