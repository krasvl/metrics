package server

import (
	"context"
	"log"
	"net/http"
	"net/http/pprof"

	"github.com/go-chi/chi/v5"

	"go.uber.org/zap"

	"metrics/internal/handlers"
	"metrics/internal/middleware"
	"metrics/internal/storage"
)

type Config struct {
	Address         string
	Key             string
	FileStoragePath string
	DatabaseDSN     string
	StoreInterval   int
	Restore         bool
	StoreFile       bool
}

type Server struct {
	storage storage.MetricsStorage
	handler *handlers.MetricsHandler
	logger  *zap.Logger
	addr    string
	key     string
	config  Config
}

func NewServer(addr string, metricsStorage storage.MetricsStorage, key string, logger *zap.Logger) *Server {
	handler := handlers.NewMetricsHandler(metricsStorage, logger)
	config := Config{
		Address: addr,
		Key:     key,
	}
	return &Server{addr: addr, key: key, storage: metricsStorage, handler: handler, logger: logger, config: config}
}

func pprofHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
	return mux
}

func (s *Server) Start(ctx context.Context) error {
	r := chi.NewRouter()

	r.Use(middleware.WithLogging(s.logger))
	r.Use(middleware.WithHashValidation(s.key))
	r.Use(middleware.WithDecompress)
	r.Use(middleware.WithCompress)
	r.Use(middleware.WithHashHeader(s.key))

	r.Mount("/debug/pprof", pprofHandler())

	r.Get("/", s.handler.GetMetricsReportHandler)

	r.Get("/ping", s.handler.PingHandler)

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

	r.Post("/updates/", s.handler.SetMetricsHandler)

	r.Route("/update", func(r chi.Router) {
		r.Post("/", s.handler.SetMetricHandler)

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

	server := &http.Server{
		Addr:    s.addr,
		Handler: r,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("Shutting down server...")

	if err := server.Shutdown(context.Background()); err != nil {
		log.Fatalf("Server shutdown failed: %v", err)
	}

	return nil
}
