package handlers

import (
	"fmt"
	"html/template"
	"net/http"
	"strconv"

	"metrics/internal/storage"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Metric represents a single metric with its type and value.
type Metric struct {
	Delta *int64   `json:"delta,omitempty"`
	Value *float64 `json:"value,omitempty"`
	ID    string   `json:"id"`
	MType string   `json:"type"`
}

// MetricsHandler handles HTTP requests for metrics operations.
type MetricsHandler struct {
	storage storage.MetricsStorage
	logger  *zap.Logger
}

// NewMetricsHandler creates a new instance of MetricsHandler.
func NewMetricsHandler(metricsStorage storage.MetricsStorage, logger *zap.Logger) *MetricsHandler {
	return &MetricsHandler{storage: metricsStorage, logger: logger}
}

// SetGaugeMetricHandler handles setting a gauge metric.
// @Summary Set Gauge Metric.
// @Description Sets a gauge metric by name and value.
// @Tags Metrics.
// @Param metricName path string true "Metric Name".
// @Param metricValue path string true "Metric Value".
// @Success 200 {string} string "OK".
// @Failure 400 {string} string "Bad Request".
// @Failure 404 {string} string "Not Found".
// @Router /update/gauge/{metricName}/{metricValue} [post].
func (h *MetricsHandler) SetGaugeMetricHandler(c *gin.Context) {
	ctx := c.Request.Context()
	metricName := c.Param("metricName")
	metricValue := c.Param("metricValue")

	if metricName == "" {
		c.String(http.StatusNotFound, "No metric name")
		return
	}
	value, err := strconv.ParseFloat(metricValue, 64)
	if err != nil {
		c.String(http.StatusBadRequest, "Gauge must be float32")
		return
	}

	if err := h.storage.SetGauge(ctx, metricName, storage.Gauge(value)); err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		h.logger.Error("cant set gauge", zap.Error(err))
		return
	}

	c.Status(http.StatusOK)
}

// SetCounterMetricHandler handles setting a counter metric.
// @Summary Set Counter Metric.
// @Description Sets a counter metric by name and value.
// @Tags Metrics.
// @Param metricName path string true "Metric Name".
// @Param metricValue path string true "Metric Value".
// @Success 200 {string} string "OK".
// @Failure 400 {string} string "Bad Request".
// @Failure 404 {string} string "Not Found".
// @Router /update/counter/{metricName}/{metricValue} [post].
func (h *MetricsHandler) SetCounterMetricHandler(c *gin.Context) {
	ctx := c.Request.Context()
	metricName := c.Param("metricName")
	metricValue := c.Param("metricValue")

	if metricName == "" {
		c.String(http.StatusNotFound, "No metric name")
		return
	}
	value, err := strconv.ParseInt(metricValue, 10, 32)
	if err != nil {
		c.String(http.StatusBadRequest, "Counter must be int32")
		return
	}

	if err := h.storage.SetCounter(ctx, metricName, storage.Counter(value)); err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		h.logger.Error("cant set counter", zap.Error(err))
		return
	}

	c.Status(http.StatusOK)
}

// GetGaugeMetricHandler handles retrieving a gauge metric.
// @Summary Get Gauge Metric.
// @Description Retrieves a gauge metric by name.
// @Tags Metrics.
// @Param metricName path string true "Metric Name".
// @Success 200 {string} string "Metric Value".
// @Failure 404 {string} string "Not Found".
// @Router /value/gauge/{metricName} [get].
func (h *MetricsHandler) GetGaugeMetricHandler(c *gin.Context) {
	ctx := c.Request.Context()
	metricName := c.Param("metricName")
	if metricName == "" {
		c.String(http.StatusNotFound, "No metric name")
		return
	}
	value, ok, err := h.storage.GetGauge(ctx, metricName)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		h.logger.Error("cant get gauge", zap.Error(err))
		return
	}

	if !ok {
		c.String(http.StatusNotFound, "No such metric")
		return
	}

	c.String(http.StatusOK, strconv.FormatFloat(float64(value), 'f', -1, 64))
}

// GetCounterMetricHandler handles retrieving a counter metric.
// @Summary Get Counter Metric.
// @Description Retrieves a counter metric by name.
// @Tags Metrics.
// @Param metricName path string true "Metric Name".
// @Success 200 {string} string "Metric Value".
// @Failure 404 {string} string "Not Found".
// @Router /value/counter/{metricName} [get].
func (h *MetricsHandler) GetCounterMetricHandler(c *gin.Context) {
	ctx := c.Request.Context()
	metricName := c.Param("metricName")
	if metricName == "" {
		c.String(http.StatusNotFound, "No metric name")
		return
	}
	value, ok, err := h.storage.GetCounter(ctx, metricName)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		h.logger.Error("cant get counter", zap.Error(err))
		return
	}

	if !ok {
		c.String(http.StatusNotFound, "No such metric")
		return
	}

	c.String(http.StatusOK, strconv.FormatInt(int64(value), 10))
}

// GetMetricsHandler handles retrieving a metric by ID.
// @Summary Get Metric.
// @Description Retrieves a metric by its ID.
// @Tags Metrics.
// @Accept json.
// @Produce json.
// @Param metric body Metric true "Metric".
// @Success 200 {object} Metric.
// @Failure 400 {string} string "Bad Request".
// @Failure 404 {string} string "Not Found".
// @Router /value [post].
func (h *MetricsHandler) GetMetricsHandler(c *gin.Context) {
	ctx := c.Request.Context()
	if c.GetHeader("Content-Type") != "application/json" {
		c.String(http.StatusUnsupportedMediaType, "Unsupported Content-Type, expected application/json")
		return
	}

	var metric Metric
	if err := c.ShouldBindJSON(&metric); err != nil {
		c.String(http.StatusBadRequest, "Bad json")
		return
	}

	switch metric.MType {
	case "gauge":
		value, ok, err := h.storage.GetGauge(ctx, metric.ID)
		if err != nil {
			c.String(http.StatusInternalServerError, err.Error())
			h.logger.Error("cant get gauge", zap.Error(err))
			return
		}
		if !ok {
			c.String(http.StatusNotFound, "No such metric")
			return
		}
		metric.Value = new(float64)
		*metric.Value = float64(value)
	case "counter":
		value, ok, err := h.storage.GetCounter(ctx, metric.ID)
		if err != nil {
			c.String(http.StatusInternalServerError, err.Error())
			h.logger.Error("cant get counter", zap.Error(err))
			return
		}
		if !ok {
			c.String(http.StatusNotFound, "No such metric")
			return
		}
		metric.Delta = new(int64)
		*metric.Delta = int64(value)
	default:
		c.String(http.StatusBadRequest, "No such metric")
		return
	}

	c.JSON(http.StatusOK, metric)
}

// SetMetricHandler handles setting a single metric.
// @Summary Set Metric.
// @Description Sets a single metric.
// @Tags Metrics.
// @Accept json.
// @Produce json.
// @Param metric body Metric true "Metric".
// @Success 200 {object} Metric.
// @Failure 400 {string} string "Bad Request".
// @Router /update [post].
func (h *MetricsHandler) SetMetricHandler(c *gin.Context) {
	ctx := c.Request.Context()
	if c.GetHeader("Content-Type") != "application/json" {
		c.String(http.StatusUnsupportedMediaType, "Unsupported Content-Type, expected application/json")
		return
	}

	var metric Metric
	if err := c.ShouldBindJSON(&metric); err != nil {
		c.String(http.StatusBadRequest, "Bad json")
		return
	}

	switch metric.MType {
	case "gauge":
		if metric.Value == nil {
			c.String(http.StatusBadRequest, "Gauge must be float32")
			return
		}
		if err := h.storage.SetGauge(ctx, metric.ID, storage.Gauge(*metric.Value)); err != nil {
			c.String(http.StatusInternalServerError, err.Error())
			h.logger.Error("cant set gauge", zap.Error(err))
			return
		}
	case "counter":
		if metric.Delta == nil {
			c.String(http.StatusBadRequest, "Counter must be int32")
			return
		}
		if err := h.storage.SetCounter(ctx, metric.ID, storage.Counter(*metric.Delta)); err != nil {
			c.String(http.StatusInternalServerError, err.Error())
			h.logger.Error("cant set counter", zap.Error(err))
			return
		}
	default:
		c.String(http.StatusBadRequest, "No such metric")
		return
	}

	c.JSON(http.StatusOK, metric)
}

// SetMetricsHandler handles setting multiple metrics in a batch.
// @Summary Set Metrics Batch.
// @Description Sets multiple metrics in a batch.
// @Tags Metrics.
// @Accept json.
// @Produce json.
// @Param metrics body []Metric true "Metrics Batch".
// @Success 200 {array} Metric.
// @Failure 400 {string} string "Bad Request".
// @Router /updates [post].
func (h *MetricsHandler) SetMetricsHandler(c *gin.Context) {
	ctx := c.Request.Context()
	h.logger.Info("Received metrics batch.",
		zap.String("content_type", c.GetHeader("Content-Type")),
		zap.String("content_encoding", c.GetHeader("Content-Encoding")),
	)

	var metrics []Metric
	if err := c.ShouldBindJSON(&metrics); err != nil {
		c.String(http.StatusBadRequest, "Bad json")
		h.logger.Error("Failed to decode metrics batch.", zap.Error(err))
		return
	}

	gaugeMetrics := make(map[string]storage.Gauge)
	counterMetrics := make(map[string]storage.Counter)

	for _, metric := range metrics {
		switch metric.MType {
		case "gauge":
			if metric.Value == nil {
				c.String(http.StatusBadRequest, fmt.Sprintf("Gauge %s must be float32.", metric.ID))
				return
			}
			gaugeMetrics[metric.ID] = storage.Gauge(*metric.Value)
		case "counter":
			if metric.Delta == nil {
				c.String(http.StatusBadRequest, fmt.Sprintf("Counter %s must be int32.", metric.ID))
				return
			}
			counterMetrics[metric.ID] += storage.Counter(*metric.Delta)
		default:
			c.String(http.StatusBadRequest, fmt.Sprintf("No metric %s type.", metric.ID))
			return
		}
	}

	if len(gaugeMetrics) > 0 {
		if err := h.storage.SetGauges(ctx, gaugeMetrics); err != nil {
			c.String(http.StatusInternalServerError, err.Error())
			h.logger.Error("cant set gauges.", zap.Error(err))
			return
		}
	}

	if len(counterMetrics) > 0 {
		if err := h.storage.SetCounters(ctx, counterMetrics); err != nil {
			c.String(http.StatusInternalServerError, err.Error())
			h.logger.Error("cant set counters.", zap.Error(err))
			return
		}
	}

	c.JSON(http.StatusOK, metrics)
}

// GetMetricsReportHandler handles generating an HTML report of all metrics.
// @Summary Get Metrics Report.
// @Description Returns an HTML report of all metrics.
// @Tags Metrics.
// @Produce html.
// @Success 200 {string} string "HTML report".
// @Router / [get].
func (h *MetricsHandler) GetMetricsReportHandler(c *gin.Context) {
	ctx := c.Request.Context()
	gauges, err := h.storage.GetGauges(ctx)
	h.logger.Info("Received gauges.",
		zap.String("gauges", fmt.Sprintf("%v", gauges)),
	)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		h.logger.Error("cant get gauges.", zap.Error(err))
		return
	}

	counters, err := h.storage.GetCounters(ctx)
	h.logger.Info("Received counters.",
		zap.String("counters", fmt.Sprintf("%v", counters)),
	)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		h.logger.Error("cant get counters.", zap.Error(err))
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

	c.Header("Content-Type", "text/html")
	if err := tmpl.Execute(c.Writer, data); err != nil {
		c.String(http.StatusInternalServerError, "Cant render report.")
		h.logger.Error("Cant render report.", zap.Error(err))
	}
}

// PingHandler handles health checks.
// @Summary Ping.
// @Description Health check endpoint.
// @Tags Health.
// @Success 200 {string} string "OK".
// @Router /ping [get].
func (h *MetricsHandler) PingHandler(c *gin.Context) {
	ctx := c.Request.Context()
	switch dbstorage := h.storage.(type) {
	case *storage.PostgresStorage:
		if err := dbstorage.Ping(ctx); err != nil {
			c.String(http.StatusInternalServerError, "cant ping db.")
			h.logger.Error("cant ping db.", zap.Error(err))
			return
		}
		c.Status(http.StatusOK)

	default:
		c.Status(http.StatusOK)
		return
	}
}
