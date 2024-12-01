package handlers

import (
	"fmt"
	"net/http"
	"strconv"

	"metrics/internal/storage"

	"github.com/go-chi/chi/v5"
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
	value, err := strconv.ParseFloat(metricValue, 64)
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

func (h *MetricsHandler) GetGaugeMetricHandler(w http.ResponseWriter, r *http.Request) {
	metricName := chi.URLParam(r, "metricName")
	if metricName == "" {
		http.Error(w, "No metric name", http.StatusNotFound)
		return
	}
	value, exist := h.storage.GetGauge(metricName)

	if !exist {
		http.Error(w, "No metric with such name", http.StatusNotFound)
		return
	}

	w.Write([]byte(fmt.Sprintf("%v", value)))
	w.WriteHeader(http.StatusOK)
}

func (h *MetricsHandler) GetCounterMetricHandler(w http.ResponseWriter, r *http.Request) {
	metricName := chi.URLParam(r, "metricName")
	if metricName == "" {
		http.Error(w, "No metric name", http.StatusNotFound)
		return
	}
	value, exist := h.storage.GetCounter(metricName)

	if !exist {
		http.Error(w, "No metric with such name", http.StatusNotFound)
		return
	}

	w.Write([]byte(fmt.Sprintf("%v", value)))
	w.WriteHeader(http.StatusOK)
}

func (h *MetricsHandler) GetAllMetricsHandler(w http.ResponseWriter, r *http.Request) {
	gauges := h.storage.GetAllGauges()
	counters := h.storage.GetAllCounters()

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte("<html><body><h1>Metrics</h1>"))

	w.Write([]byte("<h2>Gauges</h2><ul>"))
	for _, name := range gauges {
		value, _ := h.storage.GetGauge(name)
		w.Write([]byte(fmt.Sprintf("<li>%s: %v</li>", name, value)))
	}
	w.Write([]byte("</ul><h2>Counters</h2><ul>"))
	for _, name := range counters {
		value, _ := h.storage.GetCounter(name)
		w.Write([]byte(fmt.Sprintf("<li>%s: %v</li>", name, value)))
	}

	w.Write([]byte("</ul></body></html>"))
}
