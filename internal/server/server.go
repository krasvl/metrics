package server

import (
	"net/http"
	"strconv"
	"strings"

	"metrics/internal/storage"
)

type Server struct {
	storage storage.MetricsStorage
}

func NewServer(storage storage.MetricsStorage) *Server {
	return &Server{storage: storage}
}

func (s *Server) SetMetricHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Not allowed", http.StatusMethodNotAllowed)
		return
	}

	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 5 {
		http.Error(w, "Invalid Request", http.StatusNotFound)
		return
	}

	metricType := parts[2]
	metricName := parts[3]
	metricValue := parts[4]

	if metricName == "" {
		http.Error(w, "Invalid metric", http.StatusNotFound)
		return
	}

	switch metricType {
	case "gauge":
		value, err := strconv.ParseFloat(metricValue, 32)
		if err != nil {
			http.Error(w, "Gauge must be float32", http.StatusBadRequest)
			return
		}
		s.storage.SetGauge(metricName, storage.Gauge(value))

	case "counter":
		value, err := strconv.ParseInt(metricValue, 10, 32)
		if err != nil {
			http.Error(w, "Counter must be int32", http.StatusBadRequest)
			return
		}
		s.storage.SetCounter(metricName, storage.Counter(value))

	default:
		http.Error(w, "Invalid metric", http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *Server) Start() error {
	http.HandleFunc("/update/", s.SetMetricHandler)
	return http.ListenAndServe(":8080", nil)
}
