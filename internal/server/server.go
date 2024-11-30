package server

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"metrics/internal/handlers"
	"metrics/internal/storage"
)

type Server struct {
	storage storage.MetricsStorage
	handler *handlers.MetricsHandler
}

func NewServer(storage storage.MetricsStorage) *Server {
	handler := handlers.NewMetricsHandler(storage)
	return &Server{storage: storage, handler: handler}
}

func (s *Server) Start() error {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Post("/update/gauge/{metricName}/{metricValue}", s.handler.SetGaugeMetricHandler)
	r.Post("/update/counter/{metricName}/{metricValue}", s.handler.SetCounterMetricHandler)

	return http.ListenAndServe(":8080", r)
}
