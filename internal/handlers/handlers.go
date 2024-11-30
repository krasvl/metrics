package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"metrics/internal/storage"
)

type MetricsHandler struct {
	storage storage.MetricsStorage
}

func NewMetricsHandler(storage storage.MetricsStorage) *MetricsHandler {
	return &MetricsHandler{storage: storage}
}

func (h *MetricsHandler) SetMetricHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Not allowed", http.StatusMethodNotAllowed)
		return
	}

	parts := strings.Split(r.URL.Path, "/")
	metricType, metricName, metricValue := "", "", ""

	if len(parts) > 2 {
		metricType = parts[2]
	}
	if len(parts) > 3 {
		metricName = parts[3]
	}
	if len(parts) > 4 {
		metricValue = parts[4]
	}

	if metricType != "gauge" && metricType != "counter" {
		http.Error(w, "Invalid metric type", http.StatusBadRequest)
		return
	}

	if metricName == "" {
		http.Error(w, "No metric", http.StatusNotFound)
		return
	}

	switch metricType {
	case "gauge":
		value, err := strconv.ParseFloat(metricValue, 32)
		if err != nil {
			http.Error(w, "Gauge must be float32", http.StatusBadRequest)
			return
		}
		h.storage.SetGauge(metricName, storage.Gauge(value))

	case "counter":
		value, err := strconv.ParseInt(metricValue, 10, 32)
		if err != nil {
			http.Error(w, "Counter must be int32", http.StatusBadRequest)
			return
		}
		h.storage.SetCounter(metricName, storage.Counter(value))

	default:
		http.Error(w, "Invalid metric type", http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
}
