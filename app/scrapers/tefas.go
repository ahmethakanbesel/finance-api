package scrapers

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/geziyor/geziyor"
	"github.com/geziyor/geziyor/client"
)

const (
	baseUrl            = "https://www.tefas.gov.tr"
	historyEndpoint    = "/api/DB/BindHistoryInfo"
	allocationEndpoint = "/api/DB/BindHistoryAllocation"
	fundPageEndpoint   = "FonAnaliz.aspx?FonKod="
	chunkSize          = 60
	dateFormat         = "2006-01-02"
)

type PriceData struct {
	Timestamp  string `json:"TARIH"`
	Date       string
	FundCode   string  `json:"FONKODU"`
	FundName   string  `json:"FONUNVAN"`
	Price      float64 `json:"FIYAT"`
	NumShares  float64 `json:"TEDPAYSAYISI"`
	NumPeople  float64 `json:"KISISAYISI"`
	TotalWorth float64 `json:"PORTFOYBUYUKLUK"`
}

type FundData struct {
	Draw            int         `json:"draw"`
	RecordsTotal    int         `json:"recordsTotal"`
	RecordsFiltered int         `json:"recordsFiltered"`
	Data            []PriceData `json:"data"`
}

func GetTefasFundData(fundCode string, startDate string, endDate string) ([]PriceData, error) {
	if fundCode == "" {
		return nil, fmt.Errorf("fund code cannot be empty")
	}

	if startDate == "" {
		return nil, fmt.Errorf("start date cannot be empty")
	}

	if endDate == "" {
		endDate = time.Now().Format(dateFormat)
	}

	data, err := scrape(fundCode, startDate, endDate)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func scrape(fundCode string, startDate string, endDate string) ([]PriceData, error) {
	// Convert start and end dates to time.Time objects for easier manipulation
	startDateTime, err := time.Parse(dateFormat, startDate)
	if err != nil {
		return nil, fmt.Errorf("error parsing start date: %v", err)
	}

	endDateTime, err := time.Parse(dateFormat, endDate)
	if err != nil {
		return nil, fmt.Errorf("error parsing end date: %v", err)
	}

	var combinedData []PriceData

	// Split the date range into smaller chunks
	for currentStartDate := startDateTime; currentStartDate.Before(endDateTime); currentStartDate = currentStartDate.AddDate(0, 0, chunkSize) {
		// Calculate the current end date for the chunk
		currentEndDate := currentStartDate.AddDate(0, 0, chunkSize-1)
		if currentEndDate.After(endDateTime) {
			currentEndDate = endDateTime
		}

		// Perform the scrape for the current chunk and aggregate the results
		chunkData := scrapeChunk(fundCode, currentStartDate.Format(dateFormat), currentEndDate.Format(dateFormat))
		combinedData = append(combinedData, chunkData...)
	}

	//exportToCSV(combinedData)

	return combinedData, nil
}

func exportToCSV(data []PriceData) {
	sort.Slice(data, func(i, j int) bool {
		return data[i].Timestamp > data[j].Timestamp
	})

	fileName := fmt.Sprintf("./exports/%v.csv", data[0].FundCode)
	file, err := os.Create(fileName)
	if err != nil {
		fmt.Printf("Error creating CSV file: %v\n", err)
		return
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write the CSV header (if needed)
	writer.Write([]string{"Date", "Price"})

	// Write the data rows
	for _, rowData := range data {
		// Write each row of data to the CSV file
		// For example: writer.Write([]string{rowData.Field1, rowData.Field2, ...})
		writer.Write([]string{parseTimestamp(rowData.Timestamp), fmt.Sprintf("%.8f", rowData.Price)})
	}
}

// Scrape a single chunk of data
func scrapeChunk(fundCode string, startDate string, endDate string) []PriceData {
	var results []PriceData
	geziyor.NewGeziyor(&geziyor.Options{
		StartRequestsFunc: func(g *geziyor.Geziyor) {
			params := url.Values{}
			params.Add("fontip", "YAT")
			params.Add("fonkod", fundCode)
			params.Add("bastarih", startDate)
			params.Add("bittarih", endDate)
			payload := io.NopCloser(strings.NewReader(params.Encode()))

			req, _ := client.NewRequest("POST", baseUrl+historyEndpoint, payload)

			req.Header.Add("X-Requested-With", "XMLHttpRequest")
			req.Header.Add("Origin", "http://www.tefas.gov.tr")
			req.Header.Add("Referer", "http://www.tefas.gov.tr/TarihselVeriler.aspx")
			req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
			req.Header.Add("Accept", "application/json")

			g.Do(req, g.Opt.ParseFunc)
		},
		ParseFunc: func(g *geziyor.Geziyor, r *client.Response) {
			var data FundData
			err := json.Unmarshal(r.Body, &data)
			for i := range data.Data {
				data.Data[i].Date = parseTimestamp(data.Data[i].Timestamp)
			}
			if err != nil {
				fmt.Println("Error:", err)
				return
			}
			results = append(results, data.Data...)
		},
	}).Start()
	return results
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

func parseJsonPrices(g *geziyor.Geziyor, r *client.Response) {
	var data FundData
	err := json.Unmarshal(r.Body, &data)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	for _, v := range data.Data {
		g.Exports <- map[string]interface{}{
			"date":  parseTimestamp(v.Timestamp),
			"price": v.Price,
		}
	}
}
