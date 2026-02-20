package server

import (
	"net/http"

	"github.com/ahmethakanbesel/finance-api/internal/job"
	"github.com/ahmethakanbesel/finance-api/internal/price"
)

// NewHandler creates the full HTTP handler with routes and middleware.
// Exported for use in tests (e.g., httptest.NewServer).
func NewHandler(priceSvc *price.Service, jobSvc *job.Service) http.Handler {
	return newMux(priceSvc, jobSvc)
}

func newMux(priceSvc *price.Service, jobSvc *job.Service) http.Handler {
	h := &handler{
		priceSvc: priceSvc,
		jobSvc:   jobSvc,
	}

	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", h.health)
	mux.HandleFunc("GET /api/v1/sources", h.listSources)
	mux.HandleFunc("GET /api/v1/prices/{symbol}", h.getPrices)
	mux.HandleFunc("GET /api/v1/jobs", h.listJobs)
	mux.HandleFunc("GET /api/v1/jobs/{id}", h.getJob)

	// Apply middleware stack: recovery -> requestID -> logging
	var handler http.Handler = mux
	handler = logging(handler)
	handler = requestID(handler)
	handler = recovery(handler)

	return handler
}
