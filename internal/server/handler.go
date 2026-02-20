package server

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ahmethakanbesel/finance-api/internal/apperror"
	"github.com/ahmethakanbesel/finance-api/internal/job"
	"github.com/ahmethakanbesel/finance-api/internal/price"
)

const dateFormat = "2006-01-02"

type handler struct {
	priceSvc *price.Service
	jobSvc   *job.Service
}

func (h *handler) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *handler) listSources(w http.ResponseWriter, _ *http.Request) {
	sources := h.priceSvc.ListSources()
	writeJSON(w, http.StatusOK, sources)
}

func (h *handler) getPrices(w http.ResponseWriter, r *http.Request) {
	source := price.Source(r.URL.Query().Get("source"))
	if source == "" {
		writeError(w, http.StatusBadRequest, "source query parameter is required")
		return
	}

	symbol := strings.ToUpper(r.PathValue("symbol"))

	startDateStr := r.URL.Query().Get("startDate")
	if startDateStr == "" {
		writeError(w, http.StatusBadRequest, "startDate is required")
		return
	}
	startDate, err := time.Parse(dateFormat, startDateStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid startDate format, expected YYYY-MM-DD")
		return
	}

	var endDate time.Time
	if v := r.URL.Query().Get("endDate"); v != "" {
		endDate, err = time.Parse(dateFormat, v)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid endDate format, expected YYYY-MM-DD")
			return
		}
	}

	currency := price.Currency(strings.ToUpper(r.URL.Query().Get("currency")))
	if currency == "" {
		currency = price.CurrencyTRY
	}

	format := r.URL.Query().Get("format")

	req := price.GetPricesRequest{
		Source:    source,
		Symbol:    symbol,
		Currency:  currency,
		StartDate: startDate,
		EndDate:   endDate,
		Format:    format,
	}

	if appErr := req.Validate(); appErr != nil {
		writeError(w, appErr.HTTPStatus(), appErr.Message())
		return
	}

	resp, err := h.priceSvc.GetPrices(r.Context(), req)
	if err != nil {
		if ae, ok := err.(*apperror.AppError); ok {
			writeError(w, ae.HTTPStatus(), ae.Message())
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if format == "csv" {
		writeCSV(w, resp.Prices)
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

func (h *handler) getJob(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid job id")
		return
	}

	req := job.GetJobRequest{ID: id}
	if appErr := req.Validate(); appErr != nil {
		writeError(w, appErr.HTTPStatus(), appErr.Message())
		return
	}

	j, err := h.jobSvc.Get(r.Context(), req)
	if err != nil {
		if ae, ok := err.(*apperror.AppError); ok {
			writeError(w, ae.HTTPStatus(), ae.Message())
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, j)
}

func (h *handler) listJobs(w http.ResponseWriter, r *http.Request) {
	req := job.ListJobsRequest{
		Source: r.URL.Query().Get("source"),
		Symbol: r.URL.Query().Get("symbol"),
	}

	jobs, err := h.jobSvc.List(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, jobs)
}
