package agent

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"math/rand/v2"
	"net/http"
	"runtime"
	"time"

	"github.com/go-resty/resty/v2"
	"go.uber.org/zap"

	"metrics/internal/storage"
)

type Agent struct {
	storage      storage.MetricsStorage
	client       *resty.Client
	logger       *zap.Logger
	serverURL    string
	pollInterval time.Duration
	pushInterval time.Duration
}

type Metrics struct {
	Delta *int64   `json:"delta,omitempty"`
	Value *float64 `json:"value,omitempty"`
	ID    string   `json:"id"`
	MType string   `json:"type"`
}

func NewAgent(
	serverURL string,
	metricsStorage storage.MetricsStorage,
	pollInterval, pushInterval time.Duration,
	logger *zap.Logger) *Agent {
	return &Agent{
		serverURL:    serverURL,
		pollInterval: pollInterval,
		pushInterval: pushInterval,
		storage:      metricsStorage,
		client:       resty.New(),
		logger:       logger,
	}
}

func (a *Agent) pollMetrics() {
	ctx := context.Background()

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	a.pollGauge(ctx, "Alloc", storage.Gauge(m.Alloc))
	a.pollGauge(ctx, "BuckHashSys", storage.Gauge(m.BuckHashSys))
	a.pollGauge(ctx, "Frees", storage.Gauge(m.Frees))
	a.pollGauge(ctx, "GCCPUFraction", storage.Gauge(m.GCCPUFraction))
	a.pollGauge(ctx, "GCSys", storage.Gauge(m.GCSys))
	a.pollGauge(ctx, "HeapAlloc", storage.Gauge(m.HeapAlloc))
	a.pollGauge(ctx, "HeapIdle", storage.Gauge(m.HeapIdle))
	a.pollGauge(ctx, "HeapInuse", storage.Gauge(m.HeapInuse))
	a.pollGauge(ctx, "HeapObjects", storage.Gauge(m.HeapObjects))
	a.pollGauge(ctx, "HeapReleased", storage.Gauge(m.HeapReleased))
	a.pollGauge(ctx, "HeapSys", storage.Gauge(m.HeapSys))
	a.pollGauge(ctx, "LastGC", storage.Gauge(m.LastGC))
	a.pollGauge(ctx, "Lookups", storage.Gauge(m.Lookups))
	a.pollGauge(ctx, "MCacheInuse", storage.Gauge(m.MCacheInuse))
	a.pollGauge(ctx, "MCacheSys", storage.Gauge(m.MCacheSys))
	a.pollGauge(ctx, "MSpanInuse", storage.Gauge(m.MSpanInuse))
	a.pollGauge(ctx, "MSpanSys", storage.Gauge(m.MSpanSys))
	a.pollGauge(ctx, "Mallocs", storage.Gauge(m.Mallocs))
	a.pollGauge(ctx, "NextGC", storage.Gauge(m.NextGC))
	a.pollGauge(ctx, "NumForcedGC", storage.Gauge(m.NumForcedGC))
	a.pollGauge(ctx, "NumGC", storage.Gauge(m.NumGC))
	a.pollGauge(ctx, "OtherSys", storage.Gauge(m.OtherSys))
	a.pollGauge(ctx, "PauseTotalNs", storage.Gauge(m.PauseTotalNs))
	a.pollGauge(ctx, "StackInuse", storage.Gauge(m.StackInuse))
	a.pollGauge(ctx, "StackSys", storage.Gauge(m.StackSys))
	a.pollGauge(ctx, "Sys", storage.Gauge(m.Sys))
	a.pollGauge(ctx, "TotalAlloc", storage.Gauge(m.TotalAlloc))

	a.pollCounter(ctx, "PollCount", storage.Counter(1))
	a.pollGauge(ctx, "RandomValue", storage.Gauge(rand.Float64()))
}

func (a *Agent) pollGauge(ctx context.Context, name string, value storage.Gauge) {
	if err := a.storage.SetGauge(ctx, name, value); err != nil {
		a.logger.Error("can't set gauge", zap.String("gauge", name), zap.Error(err))
	}
}

func (a *Agent) pollCounter(ctx context.Context, name string, value storage.Counter) {
	if err := a.storage.SetCounter(ctx, name, value); err != nil {
		a.logger.Error("can't set gauge", zap.String("gauge", name), zap.Error(err))
	}
}

func (a *Agent) pushMetrics() {
	ctx := context.Background()

	gauges, err := a.storage.GetAllGauges(ctx)
	if err != nil {
		a.logger.Error("cant get gauges", zap.Error(err))
	}
	for name, value := range gauges {
		val := float64(value)
		metric := Metrics{
			ID:    name,
			MType: "gauge",
			Value: &val,
		}
		a.pushMetric(metric)
		if err := a.storage.ClearGauge(ctx, name); err != nil {
			a.logger.Error("cant clear gauges", zap.Error(err))
		}
	}

	counters, err := a.storage.GetAllCounters(ctx)
	if err != nil {
		a.logger.Error("cant get counters", zap.Error(err))
	}
	for name, value := range counters {
		delta := int64(value)
		metric := Metrics{
			ID:    name,
			MType: "counter",
			Delta: &delta,
		}
		a.pushMetric(metric)
		if err := a.storage.ClearCounter(ctx, name); err != nil {
			a.logger.Error("cant clear counters", zap.Error(err))
		}
	}
}

func (a *Agent) pushMetric(metric Metrics) {
	var compressed bytes.Buffer
	writer := gzip.NewWriter(&compressed)
	if err := json.NewEncoder(writer).Encode(metric); err != nil {
		a.logger.Error("cant gzip metric", zap.Error(err))
		return
	}
	if err := writer.Close(); err != nil {
		a.logger.Error("cant close gzip writer", zap.Error(err))
		return
	}

	resp, err := a.client.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Content-Encoding", "gzip").
		SetBody(compressed.Bytes()).
		Post(a.serverURL + "/update/")

	if err != nil {
		a.logger.Error("cant send metric", zap.Error(err))
		return
	}
	if resp.StatusCode() != http.StatusOK {
		a.logger.Error("bad status code", zap.Error(err))
		return
	}
}

func (a *Agent) testPing() {
	resp, err := a.client.R().Get(a.serverURL + "/ping")
	if err != nil {
		a.logger.Warn("cant ping db", zap.Error(err))
		return
	}
	if resp.StatusCode() != http.StatusOK {
		a.logger.Warn("ping db fail", zap.Error(err))
		return
	}
	a.logger.Info("ping success", zap.Error(err))
}

func (a *Agent) Start() error {
	pollTicker := time.NewTicker(a.pollInterval)
	reportTicker := time.NewTicker(a.pushInterval)

	a.testPing()

	for {
		select {
		case <-pollTicker.C:
			a.pollMetrics()
		case <-reportTicker.C:
			a.pushMetrics()
		}
	}
}
