package handlers

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"metrics/internal/storage"
)

type Metric struct {
	Delta *int64   `json:"delta,omitempty"`
	Value *float64 `json:"value,omitempty"`
	ID    string   `json:"id"`
	MType string   `json:"type"`
}

type MetricsHandler struct {
	storage storage.MetricsStorage
	logger  *zap.Logger
}

func NewMetricsHandler(metricsStorage storage.MetricsStorage, logger *zap.Logger) *MetricsHandler {
	return &MetricsHandler{storage: metricsStorage, logger: logger}
}

func (h *MetricsHandler) SetGaugeMetricHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
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

	if err := h.storage.SetGauge(ctx, metricName, storage.Gauge(value)); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		h.logger.Error("cant set gauge", zap.Error(err))
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *MetricsHandler) SetCounterMetricHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
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

	if err := h.storage.SetCounter(ctx, metricName, storage.Counter(value)); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		h.logger.Error("cant set couter", zap.Error(err))
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *MetricsHandler) GetGaugeMetricHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	metricName := chi.URLParam(r, "metricName")
	if metricName == "" {
		http.Error(w, "No metric name", http.StatusNotFound)
		return
	}
	value, ok, err := h.storage.GetGauge(ctx, metricName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		h.logger.Error("cant get gauge", zap.Error(err))
		return
	}

	if !ok {
		http.Error(w, "No such metric", http.StatusNotFound)
		return
	}

	if _, err := w.Write([]byte(strconv.FormatFloat(float64(value), 'f', -1, 64))); err != nil {
		http.Error(w, "Failed to write response", http.StatusInternalServerError)
		h.logger.Error("Failed to write response", zap.Error(err))
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *MetricsHandler) GetCounterMetricHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	metricName := chi.URLParam(r, "metricName")
	if metricName == "" {
		http.Error(w, "No metric name", http.StatusNotFound)
		return
	}
	value, ok, err := h.storage.GetCounter(ctx, metricName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		h.logger.Error("cant get counter", zap.Error(err))
		return
	}

	if !ok {
		http.Error(w, "No such metric", http.StatusNotFound)
		return
	}

	if _, err := w.Write([]byte(strconv.FormatInt(int64(value), 10))); err != nil {
		http.Error(w, "Failed to write response", http.StatusInternalServerError)
		h.logger.Error("Failed to write response", zap.Error(err))
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *MetricsHandler) GetMetricsHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if r.Header.Get("Content-Type") != "application/json" {
		http.Error(w, "Unsupported Content-Type, expected application/json", http.StatusUnsupportedMediaType)
		return
	}

	var metric Metric
	if err := json.NewDecoder(r.Body).Decode(&metric); err != nil {
		http.Error(w, "Bad json", http.StatusBadRequest)
		return
	}

	switch metric.MType {
	case "gauge":
		value, ok, err := h.storage.GetGauge(ctx, metric.ID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			h.logger.Error("cant get gauge", zap.Error(err))
			return
		}
		if !ok {
			http.Error(w, "No such metric", http.StatusNotFound)
			return
		}
		metric.Value = new(float64)
		*metric.Value = float64(value)
	case "counter":
		value, ok, err := h.storage.GetCounter(ctx, metric.ID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			h.logger.Error("cant get counter", zap.Error(err))
			return
		}
		if !ok {
			http.Error(w, "No such metric", http.StatusNotFound)
			return
		}
		metric.Delta = new(int64)
		*metric.Delta = int64(value)
	default:
		http.Error(w, "No such metric", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(metric); err != nil {
		http.Error(w, "Failed to write response", http.StatusInternalServerError)
		h.logger.Error("Failed to write response", zap.Error(err))
		return
	}
}

func (h *MetricsHandler) SetMetricHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if r.Header.Get("Content-Type") != "application/json" {
		http.Error(w, "Unsupported Content-Type, expected application/json", http.StatusUnsupportedMediaType)
		return
	}

	var metric Metric
	if err := json.NewDecoder(r.Body).Decode(&metric); err != nil {
		http.Error(w, "Bad json", http.StatusBadRequest)
		return
	}

	switch metric.MType {
	case "gauge":
		if metric.Value == nil {
			http.Error(w, "Gauge must be float32", http.StatusBadRequest)
			return
		}
		if err := h.storage.SetGauge(ctx, metric.ID, storage.Gauge(*metric.Value)); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			h.logger.Error("cant set gauge", zap.Error(err))
			return
		}
	case "counter":
		if metric.Delta == nil {
			http.Error(w, "Counter must be int32", http.StatusBadRequest)
			return
		}
		if err := h.storage.SetCounter(ctx, metric.ID, storage.Counter(*metric.Delta)); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			h.logger.Error("cant set counter", zap.Error(err))
			return
		}
	default:
		http.Error(w, "No such metric", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(metric); err != nil {
		http.Error(w, "Failed to write response", http.StatusInternalServerError)
		h.logger.Error("Failed to write response", zap.Error(err))
		return
	}
}

func (h *MetricsHandler) SetMetricsHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	h.logger.Info("Received metrics batch",
		zap.String("content_type", r.Header.Get("Content-Type")),
		zap.String("content_encoding", r.Header.Get("Content-Encoding")),
	)

	var metrics []Metric
	if err := json.NewDecoder(r.Body).Decode(&metrics); err != nil {
		http.Error(w, "Bad json", http.StatusBadRequest)
		h.logger.Error("Failed to decode metrics batch", zap.Error(err))
		return
	}

	gaugeMetrics := make(map[string]storage.Gauge)
	counterMetrics := make(map[string]storage.Counter)

	for _, metric := range metrics {
		switch metric.MType {
		case "gauge":
			if metric.Value == nil {
				http.Error(w, fmt.Sprintf("Gauge %s must be float32", metric.ID), http.StatusBadRequest)
				return
			}
			gaugeMetrics[metric.ID] = storage.Gauge(*metric.Value)
		case "counter":
			if metric.Delta == nil {
				http.Error(w, fmt.Sprintf("Counter %s must be int32", metric.ID), http.StatusBadRequest)
				return
			}
			counterMetrics[metric.ID] += storage.Counter(*metric.Delta)
		default:
			http.Error(w, fmt.Sprintf("No metric %s type", metric.ID), http.StatusBadRequest)
			return
		}
	}

	if len(gaugeMetrics) > 0 {
		if err := h.storage.SetGauges(ctx, gaugeMetrics); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			h.logger.Error("cant set gauges", zap.Error(err))
			return
		}
	}

	if len(counterMetrics) > 0 {
		if err := h.storage.SetCounters(ctx, counterMetrics); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			h.logger.Error("cant set counters", zap.Error(err))
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(metrics); err != nil {
		http.Error(w, "Failed to write response", http.StatusInternalServerError)
		h.logger.Error("Failed to write response", zap.Error(err))
		return
	}
}

func (h *MetricsHandler) GetMetricsReportHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	gauges, err := h.storage.GetGauges(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		h.logger.Error("cant get gauges", zap.Error(err))
		return
	}

	counters, err := h.storage.GetCounters(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		h.logger.Error("cant get counters", zap.Error(err))
		return
	}

	data := map[string]interface{}{
		"Gauges":   gauges,
		"Counters": counters,
	}

	tmpl := template.Must(template.New("metrics").Parse(`
<!DOCTYPE html>
<html>
<head>
    <title>Metrics</title>
</head>
<body>
    <h1>Metrics</h1>
    <h2>Gauges</h2>
    <ul>
        {{- range $name, $value := .Gauges }}
        <li>{{$name}}: {{$value}}</li>
        {{- end }}
    </ul>
    <h2>Counters</h2>
    <ul>
        {{- range $name, $value := .Counters }}
        <li>{{$name}}: {{$value}}</li>
        {{- end }}
    </ul>
</body>
</html>
	`))

	w.Header().Set("Content-Type", "text/html")
	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, "Cant render report", http.StatusInternalServerError)
		h.logger.Error("Cant render report", zap.Error(err))
	}
}

func (h *MetricsHandler) PingHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	switch dbstorage := h.storage.(type) {
	case *storage.PostgresStorage:
		if err := dbstorage.Ping(ctx); err != nil {
			http.Error(w, "cant ping db", http.StatusInternalServerError)
			h.logger.Error("cant ping db", zap.Error(err))
			return
		}
		w.WriteHeader(http.StatusOK)

	default:
		w.WriteHeader(http.StatusOK)
		return
	}
}
