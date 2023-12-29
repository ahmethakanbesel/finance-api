package scraper

import "github.com/pocketbase/pocketbase/models"

var _ models.Model = (*SymbolPrice)(nil)

type SymbolPrice struct {
	models.BaseModel
	Date  string
	Close float64
}

func (*SymbolPrice) TableName() string {
	return "prices"
}

type Scrape struct {
	models.BaseModel
	Source    string
	Symbol    string
	StartDate string
	EndDate   string
	Currency  string
}

func (*Scrape) TableName() string {
	return "scrapes"
}

type ApiResponse struct {
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}
