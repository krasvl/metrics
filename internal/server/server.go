package server

import (
	"bytes"
	"compress/gzip"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
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
	key     string
}

func NewServer(addr string, metricsStorage storage.MetricsStorage, key string, logger *zap.Logger) *Server {
	handler := handlers.NewMetricsHandler(metricsStorage, logger)
	return &Server{addr: addr, key: key, storage: metricsStorage, handler: handler, logger: logger}
}

type gzipResponseWriter struct {
	http.ResponseWriter
	Writer io.Writer
}

func (g *gzipResponseWriter) Write(b []byte) (int, error) {
	n, err := g.Writer.Write(b)
	if err != nil {
		return n, fmt.Errorf("cant gzip body: %w", err)
	}
	return n, nil
}

func (s *Server) WithDecompress(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Encoding") == "gzip" {
			gz, err := gzip.NewReader(r.Body)
			if err != nil {
				http.Error(w, "cant gzip body", http.StatusInternalServerError)
				return
			}
			defer func() {
				if err := gz.Close(); err != nil {
					http.Error(w, "cant close gzip writer", http.StatusInternalServerError)
				}
			}()
			r.Body = gz
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) WithCompress(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Accept-Encoding") == "gzip" &&
			(r.Header.Get("Accept") == "application/json" || r.Header.Get("Accept") == "text/html") {
			gz, err := gzip.NewWriterLevel(w, gzip.BestSpeed)
			if err != nil {
				http.Error(w, "cant gzip body", http.StatusInternalServerError)
				return
			}
			defer func() {
				if err := gz.Close(); err != nil {
					http.Error(w, "cant close gzip writer", http.StatusInternalServerError)
				}
			}()

			rw := &gzipResponseWriter{ResponseWriter: w, Writer: gz}

			w.Header().Set("Content-Encoding", "gzip")
			next.ServeHTTP(rw, r)
		} else {
			next.ServeHTTP(w, r)
		}
	})
}

type logResponseWriter struct {
	http.ResponseWriter
	statusCode    int
	contentLength int
}

func (lrw *logResponseWriter) WriteHeader(code int) {
	lrw.statusCode = code
	lrw.ResponseWriter.WriteHeader(code)
}

func (lrw *logResponseWriter) Write(data []byte) (int, error) {
	size, err := lrw.ResponseWriter.Write(data)
	lrw.contentLength += size
	if err != nil {
		return size, fmt.Errorf("cant log body: %w", err)
	}
	return size, nil
}

func (s *Server) WithLogging(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		rw := &logResponseWriter{ResponseWriter: w}

		h.ServeHTTP(rw, r)

		duration := time.Since(start)

		s.logger.Info("Request processed",
			zap.String("uri", r.RequestURI),
			zap.String("method", r.Method),
			zap.Duration("duration", duration),
			zap.Int("status", rw.statusCode),
			zap.Int("content_length", rw.contentLength),
		)
	})
}

type hashResponseWriter struct {
	http.ResponseWriter
	writer io.Writer
}

func (rw *hashResponseWriter) Write(p []byte) (int, error) {
	return rw.writer.Write(p)
}

func (s *Server) WithHashValidation(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.key == "" || r.Header.Get("HashSHA256") == "" {
			next.ServeHTTP(w, r)
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "cant read request body", http.StatusInternalServerError)
			return
		}
		r.Body = io.NopCloser(io.Reader(bytes.NewBuffer(body)))

		expectedHash := s.getHash(body)
		receivedHash := r.Header.Get("HashSHA256")

		if receivedHash != expectedHash {
			http.Error(w, "invalid hash", http.StatusBadRequest)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *Server) WithHashHeader(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := new(bytes.Buffer)
		mw := io.MultiWriter(w, buf)
		rw := &hashResponseWriter{ResponseWriter: w, writer: mw}

		next.ServeHTTP(rw, r)

		if s.key != "" && len(buf.Bytes()) > 0 {
			hash := s.getHash(buf.Bytes())
			w.Header().Set("HashSHA256", hash)
		}
	})
}

func (s *Server) getHash(data []byte) string {
	if s.key == "" {
		return ""
	}
	h := hmac.New(sha256.New, []byte(s.key))
	h.Write(data)
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

func (s *Server) Start() error {
	r := chi.NewRouter()

	r.Use(s.WithLogging)

	r.Use(s.WithHashValidation)

	r.Use(s.WithDecompress)
	r.Use(s.WithCompress)

	r.Use(s.WithHashHeader)

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

	if err := http.ListenAndServe(s.addr, r); err != nil {
		log.Fatalf("Server error: %v", err)
	}
	return nil
}
