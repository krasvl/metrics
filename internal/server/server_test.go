package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"metrics/internal/middleware"
	"metrics/internal/storage"

	"go.uber.org/zap/zaptest"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
)

func TestServerRoutes(t *testing.T) {
	logger := zaptest.NewLogger(t)
	mockStorage := storage.NewMemStorage()
	server := NewServer(":8080", mockStorage, "test-key", logger)

	r := chi.NewRouter()
	r.Use(middleware.WithLogging(server.logger))
	r.Use(middleware.WithHashValidation(server.key))
	r.Use(middleware.WithDecompress)
	r.Use(middleware.WithCompress)
	r.Use(middleware.WithHashHeader(server.key))

	r.Get("/", server.handler.GetMetricsReportHandler)
	r.Get("/ping", server.handler.PingHandler)

	req, _ := http.NewRequest(http.MethodGet, "/", http.NoBody)
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)
	assert.Equal(t, http.StatusOK, resp.Code)

	req, _ = http.NewRequest(http.MethodGet, "/ping", http.NoBody)
	resp = httptest.NewRecorder()
	r.ServeHTTP(resp, req)
	assert.Equal(t, http.StatusOK, resp.Code)

	req, _ = http.NewRequest(http.MethodGet, "/invalid", http.NoBody)
	resp = httptest.NewRecorder()
	r.ServeHTTP(resp, req)
	assert.Equal(t, http.StatusNotFound, resp.Code)
}
