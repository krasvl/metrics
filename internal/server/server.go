package server

import (
	"net/http"

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
	http.HandleFunc("/update/", s.handler.SetMetricHandler)
	return http.ListenAndServe(":8080", nil)
}
