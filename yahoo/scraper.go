package yahoo

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/ahmethakanbesel/finance-api/scraper"
)

const (
	downloadEndpoint = "https://query1.finance.yahoo.com/v7/finance/download/%s?%s"
	dateFormat       = "2006-01-02"
	chunkSize        = 1250
)

type Scraper struct {
	workers int
	wg      sync.WaitGroup
}

var _ scraper.Scraper = (*Scraper)(nil)

func NewScraper(options ...func(*Scraper)) *Scraper {
	s := &Scraper{}
	for _, o := range options {
		o(s)
	}
	return s
}

func WithWorkers(workers int) func(*Scraper) {
	return func(s *Scraper) {
		s.workers = workers
	}
}

type (
	chunk struct {
		symbol    string
		startDate time.Time
		endDate   time.Time
	}
)

func (s *Scraper) GetSymbolData(symbol string, startDate, endDate time.Time) (<-chan *scraper.SymbolPrice, error) {
	if symbol == "" {
		return nil, fmt.Errorf("symbol cannot be empty")
	}

	if startDate.IsZero() {
		return nil, fmt.Errorf("start date cannot be empty")
	}

	if endDate.IsZero() {
		endDate = time.Now()
	}

	if startDate.After(endDate) {
		return nil, fmt.Errorf("start date cannot be after end date")
	}

	dataChan := make(chan *scraper.SymbolPrice, chunkSize*s.workers)
	chunkChan := s.createChunks(symbol, startDate, endDate)

	for i := 0; i < s.workers; i++ {
		s.wg.Add(1)
		go s.worker(chunkChan, dataChan)
	}

	s.wg.Wait()
	close(dataChan)

	return dataChan, nil
}

func (s *Scraper) createChunks(symbol string, startDate, endDate time.Time) <-chan *chunk {
	chunkChan := make(chan *chunk)

	go func() {
		defer close(chunkChan)

		for currentStartDate := startDate; currentStartDate.Before(endDate); currentStartDate = currentStartDate.AddDate(0, 0, chunkSize) {
			currentEndDate := currentStartDate.AddDate(0, 0, chunkSize-1)
			if currentEndDate.After(endDate) {
				currentEndDate = endDate
			}

			chunkChan <- &chunk{
				symbol:    symbol,
				startDate: currentStartDate,
				endDate:   currentEndDate,
			}
		}
	}()

	return chunkChan
}

func (s *Scraper) worker(chunkChan <-chan *chunk, dataChan chan<- *scraper.SymbolPrice) {
	defer s.wg.Done()

	for chunk := range chunkChan {
		chunkData, err := getSymbolData(chunk.symbol, chunk.startDate, chunk.endDate)
		if err != nil {
			slog.Error("error retrieving tefas data", "fund", chunk.symbol, "startDate", chunk.startDate, "endDate", chunk.endDate, "error", err)
			continue
		}

		csvReader := csv.NewReader(bytes.NewReader(chunkData))
		data, err := csvReader.ReadAll()
		if err != nil {
			slog.Error("error parsing csv", "error", err)
			continue
		}

		for _, row := range data {
			if len(row) < 5 {
				continue
			}

			parsedPrice, err := strconv.ParseFloat(row[4], 64)
			if err != nil {
				continue
			}

			dataChan <- &scraper.SymbolPrice{
				Date:  row[0],
				Close: parsedPrice,
			}
		}
	}
}

func getSymbolData(symbol string, startDate, endDate time.Time) ([]byte, error) {
	client := &http.Client{}

	params := url.Values{}
	params.Add("events", "history")
	params.Add("interval", "1d")
	params.Add("corsDomain", "finance.yahoo.com")
	params.Add("period1", strconv.FormatInt(startDate.Unix(), 10))
	params.Add("period2", strconv.FormatInt(endDate.Unix(), 10))

	req, err := http.NewRequest("GET", fmt.Sprintf(downloadEndpoint, symbol, params.Encode()), nil)
	if err != nil {
		return nil, err
	}

	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	slog.Info("retrieved yahoo data", "symbol", symbol, "startDate", startDate, "endDate", endDate)
	return body, nil
}
