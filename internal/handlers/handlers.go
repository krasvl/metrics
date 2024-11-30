package handlers

import (
	"net/http"
	"strconv"

	"metrics/internal/storage"

	"github.com/go-chi/chi"
)

type MetricsHandler struct {
	storage storage.MetricsStorage
}

func NewMetricsHandler(storage storage.MetricsStorage) *MetricsHandler {
	return &MetricsHandler{storage: storage}
}

func (h *MetricsHandler) SetGaugeMetricHandler(w http.ResponseWriter, r *http.Request) {
	metricName := chi.URLParam(r, "metricName")
	metricValue := chi.URLParam(r, "metricValue")

	if metricName == "" {
		http.Error(w, "No metric name", http.StatusNotFound)
		return
	}
	value, err := strconv.ParseFloat(metricValue, 32)
	if err != nil {
		http.Error(w, "Gauge must be float32", http.StatusBadRequest)
		return
	}
	h.storage.SetGauge(metricName, storage.Gauge(value))

	w.WriteHeader(http.StatusOK)
}

func (h *MetricsHandler) SetCounterMetricHandler(w http.ResponseWriter, r *http.Request) {
	metricName := chi.URLParam(r, "metricName")
	metricValue := chi.URLParam(r, "metricValue")

	if metricName == "" {
		http.Error(w, "No metric name", http.StatusNotFound)
		return
	}
	value, err := strconv.ParseInt(metricValue, 10, 32)
	if err != nil {
		http.Error(w, "Counter must be int32", http.StatusBadRequest)
		return
	}
	h.storage.SetCounter(metricName, storage.Counter(value))

	w.WriteHeader(http.StatusOK)
}
