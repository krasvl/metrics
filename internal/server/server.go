package server

import (
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"go.uber.org/zap"

	"metrics/internal/handlers"
	"metrics/internal/storage"
)

type Server struct {
	storage storage.MetricsStorage
	handler *handlers.MetricsHandler
	logger  *zap.Logger
	addr    string
}

func NewServer(addr string, metricsStorage storage.MetricsStorage, logger *zap.Logger) *Server {
	handler := handlers.NewMetricsHandler(metricsStorage)
	return &Server{addr: addr, storage: metricsStorage, handler: handler, logger: logger}
}

type ResponseWriter struct {
	http.ResponseWriter
	statusCode    int
	contentLenght int
}

func WithLogging(logger *zap.Logger, h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		writer := &ResponseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		h.ServeHTTP(writer, r)

		duration := time.Since(start)

		logger.Info("Request processed",
			zap.String("uri", r.RequestURI),
			zap.String("method", r.Method),
			zap.Duration("duration", duration),
			zap.Int("status", writer.statusCode),
			zap.Int("content_length", writer.contentLenght),
		)
	})
}

func (s *Server) Start() error {
	r := chi.NewRouter()

	r.Use(func(next http.Handler) http.Handler {
		return WithLogging(s.logger, next)
	})

	r.Get("/", s.handler.GetMetricsReportHandler)

	r.Route("/value", func(r chi.Router) {
		r.Post("/", s.handler.GetMetricsHandler)

		r.Get("/gauge/", s.handler.GetGaugeMetricHandler)
		r.Get("/gauge/{metricName}", s.handler.GetGaugeMetricHandler)

		r.Get("/counter/", s.handler.GetCounterMetricHandler)
		r.Get("/counter/{metricName}", s.handler.GetCounterMetricHandler)

		r.NotFound(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "Invlid metric type", http.StatusBadRequest)
		})
	})

	r.Route("/update", func(r chi.Router) {
		r.Post("/", s.handler.SetMetricsHandler)

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

	if err := http.ListenAndServe(s.addr, r); err != nil {
		log.Fatalf("Server error: %v", err)
	}
	return nil
}
