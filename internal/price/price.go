package price

import "time"

type Source string

const (
	SourceTefas     Source = "tefas"
	SourceYahoo     Source = "yahoo"
	SourceIsyatirim Source = "isyatirim"
)

type Currency string

const (
	CurrencyTRY Currency = "TRY"
	CurrencyUSD Currency = "USD"
)

type Price struct {
	ID         int64     `json:"id"`
	Source     Source    `json:"source"`
	Symbol     string    `json:"symbol"`
	Date       time.Time `json:"date"`
	ClosePrice float64   `json:"closePrice"`
	Currency   Currency  `json:"currency"`
	CreatedAt  time.Time `json:"createdAt"`
}
