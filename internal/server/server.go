package server

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/ahmethakanbesel/finance-api/internal/job"
	"github.com/ahmethakanbesel/finance-api/internal/price"
)

type Server struct {
	srv *http.Server
}

// New creates a server. The baseCtx is used as the base context for all
// incoming requests (via BaseContext). Cancelling it causes in-flight scraper
// workers to stop promptly during graceful shutdown.
func New(baseCtx context.Context, port string, priceSvc *price.Service, jobSvc *job.Service) *Server {
	return &Server{
		srv: &http.Server{
			Addr:    fmt.Sprintf(":%s", port),
			Handler: newMux(priceSvc, jobSvc),
			BaseContext: func(_ net.Listener) context.Context {
				return baseCtx
			},
			ReadTimeout:  15 * time.Second,
			WriteTimeout: 60 * time.Second,
			IdleTimeout:  120 * time.Second,
		},
	}
}

func (s *Server) Start() error {
	slog.Info("starting server", "addr", s.srv.Addr)
	return s.srv.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	slog.Info("shutting down server")
	return s.srv.Shutdown(ctx)
}
