package handlers

import (
	"context"
	"encoding/json"
	"html/template"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"metrics/internal/storage"
)

type Metrics struct {
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

func (h *MetricsHandler) SetMetricsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Content-Type") != "application/json" {
		http.Error(w, "Unsupported Content-Type, expected application/json", http.StatusUnsupportedMediaType)
		return
	}

	var metric Metrics
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
		h.storage.SetGauge(metric.ID, storage.Gauge(*metric.Value))
	case "counter":
		if metric.Delta == nil {
			http.Error(w, "Counter must be int32", http.StatusBadRequest)
			return
		}
		h.storage.SetCounter(metric.ID, storage.Counter(*metric.Delta))
	default:
		http.Error(w, "No such metric", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(metric); err != nil {
		http.Error(w, "Bad json", http.StatusInternalServerError)
		return
	}
}

func (h *MetricsHandler) GetGaugeMetricHandler(w http.ResponseWriter, r *http.Request) {
	metricName := chi.URLParam(r, "metricName")
	if metricName == "" {
		http.Error(w, "No metric name", http.StatusNotFound)
		return
	}
	value, ok := h.storage.GetGauge(metricName)

	if !ok {
		http.Error(w, "No such metric", http.StatusNotFound)
		return
	}

	if _, err := w.Write([]byte(strconv.FormatFloat(float64(value), 'f', -1, 64))); err != nil {
		http.Error(w, "Gauge must be float32", http.StatusNotFound)
	}
	w.WriteHeader(http.StatusOK)
}

func (h *MetricsHandler) GetCounterMetricHandler(w http.ResponseWriter, r *http.Request) {
	metricName := chi.URLParam(r, "metricName")
	if metricName == "" {
		http.Error(w, "No metric name", http.StatusNotFound)
		return
	}
	value, ok := h.storage.GetCounter(metricName)

	if !ok {
		http.Error(w, "No such metric", http.StatusNotFound)
		return
	}

	if _, err := w.Write([]byte(strconv.FormatInt(int64(value), 10))); err != nil {
		http.Error(w, "Counter must be int32", http.StatusNotFound)
	}
	w.WriteHeader(http.StatusOK)
}

func (h *MetricsHandler) GetMetricsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Content-Type") != "application/json" {
		http.Error(w, "Unsupported Content-Type, expected application/json", http.StatusUnsupportedMediaType)
		return
	}

	var metric Metrics
	if err := json.NewDecoder(r.Body).Decode(&metric); err != nil {
		http.Error(w, "Bad json", http.StatusBadRequest)
		return
	}

	switch metric.MType {
	case "gauge":
		value, ok := h.storage.GetGauge(metric.ID)
		if !ok {
			http.Error(w, "No such metric", http.StatusNotFound)
			return
		}
		metric.Value = new(float64)
		*metric.Value = float64(value)
	case "counter":
		value, ok := h.storage.GetCounter(metric.ID)
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
		http.Error(w, "Bad json", http.StatusInternalServerError)
		return
	}
}

func (h *MetricsHandler) GetMetricsReportHandler(w http.ResponseWriter, r *http.Request) {
	gauges := h.storage.GetAllGauges()
	counters := h.storage.GetAllCounters()

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
	}
}

func (h *MetricsHandler) PingHandler(w http.ResponseWriter, r *http.Request) {
	switch dbstorage := h.storage.(type) {
	case *storage.PostgresStorage:
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		if err := dbstorage.Ping(ctx); err != nil {
			http.Error(w, "cant ping db", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)

	case *storage.MemStorage:
		http.Error(w, "server use memory storage", http.StatusInternalServerError)
		return

	case *storage.FileStorage:
		http.Error(w, "server use file storage", http.StatusInternalServerError)
		return

	default:
		http.Error(w, "server use unknown storage", http.StatusInternalServerError)
		return
	}
}
