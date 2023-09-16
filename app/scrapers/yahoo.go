package scrapers

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/geziyor/geziyor"
	"github.com/geziyor/geziyor/client"
)

const (
	yahooBaseUrl          = "https://query1.finance.yahoo.com/v7/finance"
	yahooDownloadEndpoint = "/download/%s?%s"
	yahooChunkSize        = 1250
	yahooUserAgent        = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/116.0.0.0 Safari/537.36"
)

type SymbolData struct {
	Date     string
	Open     string
	High     string
	Low      string
	Close    string
	AdjClose string
	Volume   string
}

func GetYahooSymbolData(symbol string, startDate string, endDate string) ([]SymbolData, error) {
	if symbol == "" {
		return nil, fmt.Errorf("symbol cannot be empty")
	}

	if startDate == "" {
		return nil, fmt.Errorf("start date cannot be empty")
	}

	if endDate == "" {
		endDate = time.Now().Format(dateFormat)
	}

	data, err := scrapeYahoo(symbol, startDate, endDate)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func scrapeYahoo(symbol string, startDate string, endDate string) ([]SymbolData, error) {
	// Convert start and end dates to time.Time objects for easier manipulation
	startDateTime, err := time.Parse(dateFormat, startDate)
	if err != nil {
		return nil, fmt.Errorf("error parsing start date: %v", err)
	}

	endDateTime, err := time.Parse(dateFormat, endDate)
	if err != nil {
		return nil, fmt.Errorf("error parsing end date: %v", err)
	}

	var combinedData []SymbolData

	// Split the date range into smaller chunks
	for currentStartDate := startDateTime; currentStartDate.Before(endDateTime); currentStartDate = currentStartDate.AddDate(0, 0, yahooChunkSize) {
		// Calculate the current end date for the chunk
		currentEndDate := currentStartDate.AddDate(0, 0, yahooChunkSize-1)
		if currentEndDate.After(endDateTime) {
			currentEndDate = endDateTime
		}

		// Perform the scrape for the current chunk and aggregate the results
		chunkData := scrapeYahooChunk(symbol, currentStartDate.Unix(), currentEndDate.Unix())
		combinedData = append(combinedData, chunkData...)
	}

	//exportToCSV(combinedData)

	return combinedData, nil
}

func scrapeYahooChunk(symbol string, startDate int64, endDate int64) []SymbolData {
	var results []SymbolData
	geziyor.NewGeziyor(&geziyor.Options{
		RobotsTxtDisabled: true,
		StartRequestsFunc: func(g *geziyor.Geziyor) {
			params := url.Values{}
			params.Add("events", "history")
			params.Add("interval", "1d")
			params.Add("corsDomain", "finance.yahoo.com")
			params.Add("period1", strconv.FormatInt(startDate, 10))
			params.Add("period2", strconv.FormatInt(endDate, 10))

			fmt.Println(fmt.Sprintf(yahooBaseUrl+yahooDownloadEndpoint, symbol, params.Encode()))
			req, _ := client.NewRequest("GET", fmt.Sprintf(yahooBaseUrl+yahooDownloadEndpoint, symbol, params.Encode()), nil)
			req.Header.Add("User-Agent", yahooUserAgent)

			g.Do(req, g.Opt.ParseFunc)
		},
		ParseFunc: func(g *geziyor.Geziyor, r *client.Response) {
			reader := csv.NewReader(bytes.NewReader(r.Body))
			isFirstLine := true
			for {
				// Read one line from the CSV
				record, err := reader.Read()
				if err != nil {
					break // End of file or error, break the loop
				}

				if isFirstLine {
					isFirstLine = false
					continue
				}

				row := SymbolData{
					Date:     record[0],
					Open:     record[1],
					High:     record[2],
					Low:      record[3],
					Close:    record[4],
					AdjClose: record[5],
					Volume:   record[6],
				}

				results = append(results, row)
			}
		},
	}).Start()
	return results
}
