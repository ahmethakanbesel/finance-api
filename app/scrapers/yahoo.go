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
)

type YahooScraper struct{}

var _ Scraper = (*YahooScraper)(nil)

func (y *YahooScraper) GetSymbolData(symbol string, startDate string, endDate string) ([]SymbolPrice, error) {
	if symbol == "" {
		return nil, fmt.Errorf("symbol cannot be empty")
	}

	if startDate == "" {
		return nil, fmt.Errorf("start date cannot be empty")
	}

	if endDate == "" {
		endDate = time.Now().Format(dateFormat)
	}

	data, err := scrape(y, symbol, startDate, endDate)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func (y *YahooScraper) scrapeChunk(symbol string, startDate time.Time, endDate time.Time) ([]SymbolPrice, error) {
	var results []SymbolPrice
	geziyor.NewGeziyor(&geziyor.Options{
		RobotsTxtDisabled: true,
		StartRequestsFunc: func(g *geziyor.Geziyor) {
			params := url.Values{}
			params.Add("events", "history")
			params.Add("interval", "1d")
			params.Add("corsDomain", "finance.yahoo.com")
			params.Add("period1", strconv.FormatInt(startDate.Unix(), 10))
			params.Add("period2", strconv.FormatInt(endDate.Unix(), 10))

			fmt.Println(fmt.Sprintf(yahooBaseUrl+yahooDownloadEndpoint, symbol, params.Encode()))
			req, _ := client.NewRequest("GET", fmt.Sprintf(yahooBaseUrl+yahooDownloadEndpoint, symbol, params.Encode()), nil)

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

				parsedPrice, err := strconv.ParseFloat(record[4], 64)
				if err != nil {
					continue
				}
				
				row := SymbolPrice{
					Date:  record[0],
					Close: parsedPrice,
				}

				results = append(results, row)
			}
		},
	}).Start()
	return results, nil
}

func (y *YahooScraper) getChunkSize() int {
	return yahooChunkSize
}
