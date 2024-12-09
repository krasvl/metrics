package handlers

import (
	"html/template"
	"net/http"
	"strconv"

	"metrics/internal/storage"

	"github.com/go-chi/chi/v5"
)

const metricNameKey string = "metricName"
const metricValueKey string = "metricValue"

const noMetricNameMsg string = "No metric name"
const noMetricMsg string = "No such metric"
const cantParseGaugeMsg string = "Gauge must be float32"
const cantParseCounterMsg string = "Counter must be int32"
const cantParseStatisticsMsg string = "Cant render statistics"

type MetricsHandler struct {
	storage storage.MetricsStorage
}

func NewMetricsHandler(metricsStorage storage.MetricsStorage) *MetricsHandler {
	return &MetricsHandler{storage: metricsStorage}
}

func (h *MetricsHandler) SetGaugeMetricHandler(w http.ResponseWriter, r *http.Request) {
	metricName := chi.URLParam(r, metricNameKey)
	metricValue := chi.URLParam(r, metricValueKey)

	if metricName == "" {
		http.Error(w, noMetricNameMsg, http.StatusNotFound)
		return
	}
	value, err := strconv.ParseFloat(metricValue, 64)
	if err != nil {
		http.Error(w, cantParseGaugeMsg, http.StatusBadRequest)
		return
	}
	h.storage.SetGauge(metricName, storage.Gauge(value))

	w.WriteHeader(http.StatusOK)
}

func (h *MetricsHandler) SetCounterMetricHandler(w http.ResponseWriter, r *http.Request) {
	metricName := chi.URLParam(r, metricNameKey)
	metricValue := chi.URLParam(r, metricValueKey)

	if metricName == "" {
		http.Error(w, noMetricNameMsg, http.StatusNotFound)
		return
	}
	value, err := strconv.ParseInt(metricValue, 10, 32)
	if err != nil {
		http.Error(w, cantParseCounterMsg, http.StatusBadRequest)
		return
	}
	h.storage.SetCounter(metricName, storage.Counter(value))

	w.WriteHeader(http.StatusOK)
}

func (h *MetricsHandler) GetGaugeMetricHandler(w http.ResponseWriter, r *http.Request) {
	metricName := chi.URLParam(r, metricNameKey)
	if metricName == "" {
		http.Error(w, noMetricNameMsg, http.StatusNotFound)
		return
	}
	value, ok := h.storage.GetGauge(metricName)

	if !ok {
		http.Error(w, noMetricMsg, http.StatusNotFound)
		return
	}

	if _, err := w.Write([]byte(strconv.FormatFloat(float64(value), 'f', -1, 64))); err != nil {
		http.Error(w, cantParseGaugeMsg, http.StatusNotFound)
	}
	w.WriteHeader(http.StatusOK)
}

func (h *MetricsHandler) GetCounterMetricHandler(w http.ResponseWriter, r *http.Request) {
	metricName := chi.URLParam(r, metricNameKey)
	if metricName == "" {
		http.Error(w, noMetricNameMsg, http.StatusNotFound)
		return
	}
	value, ok := h.storage.GetCounter(metricName)

	if !ok {
		http.Error(w, noMetricMsg, http.StatusNotFound)
		return
	}

	if _, err := w.Write([]byte(strconv.FormatInt(int64(value), 10))); err != nil {
		http.Error(w, cantParseCounterMsg, http.StatusNotFound)
	}
	w.WriteHeader(http.StatusOK)
}

func (h *MetricsHandler) GetAllMetricsHandler(w http.ResponseWriter, r *http.Request) {
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
		http.Error(w, cantParseStatisticsMsg, http.StatusInternalServerError)
	}
}
