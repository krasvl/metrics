package server

import (
	"context"
	"log"
	"net/http"
	"net/http/pprof"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"go.uber.org/zap"

	_ "metrics/docs"

	"metrics/internal/handlers"
	"metrics/internal/middleware"
	"metrics/internal/storage"
)

// Config holds the server configuration parameters.
type Config struct {
	Address         string
	Key             string
	FileStoragePath string
	DatabaseDSN     string
	StoreInterval   int
	Restore         bool
	StoreFile       bool
}

// Server represents the HTTP server for the metrics service.
type Server struct {
	storage storage.MetricsStorage
	handler *handlers.MetricsHandler
	logger  *zap.Logger
	config  Config
}

// NewServer creates a new instance of the Server.
// @title Metrics API
// @version 1.0
// @description API documentation for the Metrics service.
// @host localhost:8080
// @BasePath /.
func NewServer(metricsStorage storage.MetricsStorage, logger *zap.Logger, config *Config) *Server {
	handler := handlers.NewMetricsHandler(metricsStorage, logger)
	return &Server{storage: metricsStorage, handler: handler, logger: logger, config: *config}
}

func registerPprofRoutes(router *gin.Engine) {
	pprofGroup := router.Group("/debug/pprof")
	pprofGroup.GET("/", gin.WrapH(http.HandlerFunc(pprof.Index)))
	pprofGroup.GET("/cmdline", gin.WrapH(http.HandlerFunc(pprof.Cmdline)))
	pprofGroup.GET("/profile", gin.WrapH(http.HandlerFunc(pprof.Profile)))
	pprofGroup.GET("/symbol", gin.WrapH(http.HandlerFunc(pprof.Symbol)))
	pprofGroup.GET("/trace", gin.WrapH(http.HandlerFunc(pprof.Trace)))
}

// Start starts the HTTP server and listens for incoming requests.
// @title Start Server
// @description Starts the HTTP server with all routes and middleware.
func (s *Server) Start(ctx context.Context) error {
	router := gin.Default()

	router.Use(middleware.WithLogging(s.logger))
	router.Use(middleware.WithHashValidation(s.config.Key))
	router.Use(middleware.WithDecompress())
	router.Use(middleware.WithCompress())
	router.Use(middleware.WithHashHeader(s.config.Key))

	// Pprof routes
	registerPprofRoutes(router)

	// Swagger documentation route
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	router.GET("/", s.handler.GetMetricsReportHandler)

	router.GET("/ping", s.handler.PingHandler)

	router.POST("/value", s.handler.GetMetricsHandler)

	router.POST("/update/gauge/:metricName/:metricValue", s.handler.SetGaugeMetricHandler)

	router.POST("/update/counter/:metricName/:metricValue", s.handler.SetCounterMetricHandler)

	router.POST("/updates", s.handler.SetMetricsHandler)

	router.POST("/update", s.handler.SetMetricHandler)

	server := &http.Server{
		Addr:    s.config.Address,
		Handler: router,
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
