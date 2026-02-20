package price

import (
	"time"

	"github.com/ahmethakanbesel/finance-api/internal/apperror"
	"github.com/ahmethakanbesel/finance-api/internal/job"
)

type GetPricesRequest struct {
	Source    Source
	Symbol    string
	Currency  Currency
	StartDate time.Time
	EndDate   time.Time
	Format    string // "json" or "csv"
}

func (r GetPricesRequest) Validate() *apperror.AppError {
	if len(r.Symbol) < 2 {
		return apperror.New(apperror.BadRequest, "symbol must be at least 2 characters")
	}
	if r.StartDate.IsZero() {
		return apperror.New(apperror.BadRequest, "startDate is required")
	}
	if !r.EndDate.IsZero() && r.EndDate.Before(r.StartDate) {
		return apperror.New(apperror.BadRequest, "endDate must be after startDate")
	}
	if r.Currency != CurrencyTRY && r.Currency != CurrencyUSD {
		return apperror.New(apperror.BadRequest, "currency must be TRY or USD")
	}
	if r.Format != "" && r.Format != "json" && r.Format != "csv" {
		return apperror.New(apperror.BadRequest, "format must be json or csv")
	}
	return nil
}

type PricePoint struct {
	Symbol         string    `json:"symbol"`
	Date           time.Time `json:"date"`
	ClosePrice     float64   `json:"closePrice"`
	Currency       Currency  `json:"currency"`
	NativePrice    float64   `json:"nativePrice"`
	NativeCurrency Currency  `json:"nativeCurrency"`
	Rate           float64   `json:"rate"`
	Source         Source    `json:"source"`
}

type GetPricesResponse struct {
	Prices []PricePoint `json:"prices"`
	Job    *job.Job     `json:"job,omitempty"`
}
