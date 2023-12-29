package tefas

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ahmethakanbesel/finance-api/scraper"
)

const (
	baseUrl            = "https://www.tefas.gov.tr"
	historyEndpoint    = "https://www.tefas.gov.tr/api/DB/BindHistoryInfo"
	allocationEndpoint = "https://www.tefas.gov.tr/api/DB/BindHistoryAllocation"
	fundPageEndpoint   = "FonAnaliz.aspx?FonKod="
	dateFormat         = "2006-01-02"
	chunkSize          = 60
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
	tefasPriceData struct {
		Timestamp  string  `json:"TARIH"`
		FundCode   string  `json:"FONKODU"`
		FundName   string  `json:"FONUNVAN"`
		Price      float64 `json:"FIYAT"`
		NumShares  float64 `json:"TEDPAYSAYISI"`
		NumPeople  float64 `json:"KISISAYISI"`
		TotalWorth float64 `json:"PORTFOYBUYUKLUK"`
	}
	fundData struct {
		Draw            int              `json:"draw"`
		RecordsTotal    int              `json:"recordsTotal"`
		RecordsFiltered int              `json:"recordsFiltered"`
		Data            []tefasPriceData `json:"data"`
	}
	chunk struct {
		symbol    string
		startDate time.Time
		endDate   time.Time
	}
)

func (s *Scraper) GetSymbolData(symbol string, startDate, endDate time.Time) (<-chan *scraper.SymbolPrice, error) {
	if symbol == "" {
		return nil, fmt.Errorf("fund code cannot be empty")
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
		chunkData, err := getFundData(chunk.symbol, chunk.startDate, chunk.endDate)
		if err != nil {
			slog.Error("error retrieving tefas data", "fund", chunk.symbol, "startDate", chunk.startDate, "endDate", chunk.endDate, "error", err)
			continue
		}

		for i := range chunkData.Data {
			dataChan <- &scraper.SymbolPrice{
				Date:  parseTimestamp(chunkData.Data[i].Timestamp),
				Close: chunkData.Data[i].Price,
			}
		}
	}
}

func getFundData(fundCode string, startDate, endDate time.Time) (*fundData, error) {
	client := &http.Client{}

	params := url.Values{}
	params.Add("fontip", "YAT")
	params.Add("fonkod", fundCode)
	params.Add("bastarih", startDate.Format(scraper.DateFormat))
	params.Add("bittarih", endDate.Format(scraper.DateFormat))

	payload := strings.NewReader(params.Encode())
	req, err := http.NewRequest("POST", historyEndpoint, payload)
	if err != nil {
		return nil, err
	}

	req.Header.Add("X-Requested-With", "XMLHttpRequest")
	req.Header.Add("Origin", "http://www.tefas.gov.tr")
	req.Header.Add("Referer", "http://www.tefas.gov.tr/TarihselVeriler.aspx")
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Accept", "application/json")

	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	response := &fundData{}
	err = json.Unmarshal(body, response)
	if err != nil {
		return nil, err
	}

	slog.Info("retrieved tefas data", "fund", fundCode, "startDate", startDate, "endDate", endDate)
	return response, nil
}

func parseTimestamp(timestamp string) string {
	timestampInt, _ := strconv.ParseInt(timestamp, 10, 64)

	// Convert the timestamp to time.Time
	seconds := timestampInt / 1000
	nanoseconds := (timestampInt % 1000) * 1000000
	tm := time.Unix(seconds, nanoseconds)

	// YYYY-MM-DD
	formattedDate := tm.Format(dateFormat)
	return formattedDate
}
