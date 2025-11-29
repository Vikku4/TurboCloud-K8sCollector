package httpserver

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/turbocloud-io/k8s-collector/internal/core"
	tclog "github.com/turbocloud-io/k8s-collector/internal/log"
)

type Server struct {
	svc            *core.Service
	requestTimeout time.Duration
	logger         *tclog.Logger
}

func NewServer(svc *core.Service, timeout time.Duration, logger *tclog.Logger) *Server {
	return &Server{
		svc:            svc,
		requestTimeout: timeout,
		logger:         logger,
	}
}

func (s *Server) Router() http.Handler {
	r := chi.NewRouter()

	// Middlewares
	r.Use(RequestLogger(s.logger))
	r.Use(Recoverer(s.logger))

	// API routes
	r.Route("/api/v1", func(r chi.Router) {
		r.Post("/k8s/metrics/ingest", s.handleIngestMetrics)
		r.Get("/k8s/metrics/summary", s.handleGetSummary)
	})

	return r
}
