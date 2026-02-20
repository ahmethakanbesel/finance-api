package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/ahmethakanbesel/finance-api/internal/price"
)

type APIResponse[T any] struct {
	Message string `json:"message"`
	Data    T      `json:"data"`
}

func writeJSON[T any](w http.ResponseWriter, status int, data T) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(APIResponse[T]{
		Message: "ok",
		Data:    data,
	})
}

func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(APIResponse[string]{
		Message: message,
		Data:    "",
	})
}

func writeCSV(w http.ResponseWriter, prices []price.PricePoint) {
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment; filename=prices.csv")
	w.WriteHeader(http.StatusOK)

	_, _ = fmt.Fprintln(w, "Symbol,Date,Currency,Source,Close,NativePrice,NativeCurrency,Rate")
	for _, p := range prices {
		_, _ = fmt.Fprintf(w, "%s,%s,%s,%s,%.6f,%.6f,%s,%.6f\n", //nolint:gosec // CSV output from internal domain types, not user input
			p.Symbol,
			p.Date.Format(time.DateOnly),
			p.Currency,
			p.Source,
			p.ClosePrice,
			p.NativePrice,
			p.NativeCurrency,
			p.Rate,
		)
	}
}
