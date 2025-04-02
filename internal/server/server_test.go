package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"metrics/internal/middleware"
	"metrics/internal/storage"

	"go.uber.org/zap/zaptest"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestServerRoutes(t *testing.T) {
	logger := zaptest.NewLogger(t)
	mockStorage := storage.NewMemStorage()
	config := Config{
		Address: ":8080",
		Key:     "test-key",
	}
	server := NewServer(mockStorage, logger, &config)

	router := gin.Default()
	router.Use(middleware.WithLogging(server.logger))
	router.Use(middleware.WithHashValidation(server.config.Key))
	router.Use(middleware.WithDecompress())
	router.Use(middleware.WithCompress())
	router.Use(middleware.WithHashHeader(server.config.Key))

	registerPprofRoutes(router)

	router.GET("/", server.handler.GetMetricsReportHandler)
	router.GET("/ping", server.handler.PingHandler)

	req, _ := http.NewRequest(http.MethodGet, "/", http.NoBody)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	assert.Equal(t, http.StatusOK, resp.Code)

	req, _ = http.NewRequest(http.MethodGet, "/ping", http.NoBody)
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	assert.Equal(t, http.StatusOK, resp.Code)

	req, _ = http.NewRequest(http.MethodGet, "/debug/pprof/", http.NoBody)
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	assert.Equal(t, http.StatusOK, resp.Code)

	req, _ = http.NewRequest(http.MethodGet, "/invalid", http.NoBody)
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	assert.Equal(t, http.StatusNotFound, resp.Code)
}
