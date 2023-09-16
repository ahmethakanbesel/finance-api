package scrapers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/geziyor/geziyor"
	"github.com/geziyor/geziyor/client"
)

const (
	tefasBaseUrl            = "https://www.tefas.gov.tr"
	tefasHistoryEndpoint    = "/api/DB/BindHistoryInfo"
	tefasAllocationEndpoint = "/api/DB/BindHistoryAllocation"
	tefasFundPageEndpoint   = "FonAnaliz.aspx?FonKod="
	tefasChunkSize          = 60
)

type TefasScraper struct {}

var _ Scraper = (*TefasScraper)(nil)

type tefasPriceData struct {
	Timestamp  string  `json:"TARIH"`
	FundCode   string  `json:"FONKODU"`
	FundName   string  `json:"FONUNVAN"`
	Price      float64 `json:"FIYAT"`
	NumShares  float64 `json:"TEDPAYSAYISI"`
	NumPeople  float64 `json:"KISISAYISI"`
	TotalWorth float64 `json:"PORTFOYBUYUKLUK"`
}

type fundData struct {
	Draw            int              `json:"draw"`
	RecordsTotal    int              `json:"recordsTotal"`
	RecordsFiltered int              `json:"recordsFiltered"`
	Data            []tefasPriceData `json:"data"`
}

func (t *TefasScraper) GetSymbolData(symbol string, startDate string, endDate string) ([]SymbolPrice, error) {
	if symbol == "" {
		return nil, fmt.Errorf("fund code cannot be empty")
	}

	if startDate == "" {
		return nil, fmt.Errorf("start date cannot be empty")
	}

	if endDate == "" {
		endDate = time.Now().Format(dateFormat)
	}

	data, err := scrape(t, symbol, startDate, endDate)
	if err != nil {
		return nil, err
	}

	return data, nil
}

// Scrape a single chunk of data
func (t *TefasScraper) scrapeChunk(fundCode string, startDate time.Time, endDate time.Time) ([]SymbolPrice, error) {
	var results []SymbolPrice
	geziyor.NewGeziyor(&geziyor.Options{
		StartRequestsFunc: func(g *geziyor.Geziyor) {
			params := url.Values{}
			params.Add("fontip", "YAT")
			params.Add("fonkod", fundCode)
			params.Add("bastarih", startDate.Format(dateFormat))
			params.Add("bittarih", endDate.Format(dateFormat))

			payload := io.NopCloser(strings.NewReader(params.Encode()))

			req, _ := client.NewRequest("POST", tefasBaseUrl+tefasHistoryEndpoint, payload)
			req.Header.Add("X-Requested-With", "XMLHttpRequest")
			req.Header.Add("Origin", "http://www.tefas.gov.tr")
			req.Header.Add("Referer", "http://www.tefas.gov.tr/TarihselVeriler.aspx")
			req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
			req.Header.Add("Accept", "application/json")

			g.Do(req, g.Opt.ParseFunc)
		},
		ParseFunc: func(g *geziyor.Geziyor, r *client.Response) {
			var data fundData
			err := json.Unmarshal(r.Body, &data)
			for i := range data.Data {
				results = append(results, SymbolPrice{
					Date:  t.parseTimestamp(data.Data[i].Timestamp),
					Close: data.Data[i].Price,
				})
			}
			if err != nil {
				fmt.Println("Error:", err)
				return
			}
		},
	}).Start()
	return results, nil
}

func (t *TefasScraper) parseTimestamp(timestamp string) string {
	timestampInt, _ := strconv.ParseInt(timestamp, 10, 64)

	// Convert the timestamp to time.Time
	seconds := timestampInt / 1000
	nanoseconds := (timestampInt % 1000) * 1000000
	tm := time.Unix(seconds, nanoseconds)

	// YYYY-MM-DD
	formattedDate := tm.Format(dateFormat)
	return formattedDate
}

func (t *TefasScraper) getChunkSize() int {
	return tefasChunkSize
}
