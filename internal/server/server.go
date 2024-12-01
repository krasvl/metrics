package server

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"metrics/internal/handlers"
	"metrics/internal/storage"
)

type Server struct {
	addr    string
	storage storage.MetricsStorage
	handler *handlers.MetricsHandler
}

func NewServer(addr string, storage storage.MetricsStorage) *Server {
	handler := handlers.NewMetricsHandler(storage)
	return &Server{addr: addr, storage: storage, handler: handler}
}

func (s *Server) Start() error {
	r := chi.NewRouter()
	r.Use(middleware.Logger)

	r.Get("/", s.handler.GetAllMetricsHandler)

	r.Route("/value", func(r chi.Router) {
		r.Get("/gauge/", s.handler.GetGaugeMetricHandler)
		r.Get("/gauge/{metricName}", s.handler.GetGaugeMetricHandler)

		r.Get("/counter/", s.handler.GetCounterMetricHandler)
		r.Get("/counter/{metricName}", s.handler.GetCounterMetricHandler)

		r.NotFound(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "Invlid metric type", http.StatusBadRequest)
		})
	})

	r.Route("/update", func(r chi.Router) {
		r.Post("/gauge/", s.handler.SetGaugeMetricHandler)
		r.Post("/gauge/{metricName}/", s.handler.SetGaugeMetricHandler)
		r.Post("/gauge/{metricName}/{metricValue}", s.handler.SetGaugeMetricHandler)

		r.Post("/counter/", s.handler.SetCounterMetricHandler)
		r.Post("/counter/{metricName}/", s.handler.SetCounterMetricHandler)
		r.Post("/counter/{metricName}/{metricValue}", s.handler.SetCounterMetricHandler)

		r.NotFound(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "Invalid metric type", http.StatusBadRequest)
		})
	})

	return http.ListenAndServe(s.addr, r)
}
