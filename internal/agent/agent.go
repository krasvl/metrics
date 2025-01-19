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

type Metric struct {
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

	gauges := map[string]storage.Gauge{
		"Alloc":         storage.Gauge(m.Alloc),
		"BuckHashSys":   storage.Gauge(m.BuckHashSys),
		"Frees":         storage.Gauge(m.Frees),
		"GCCPUFraction": storage.Gauge(m.GCCPUFraction),
		"GCSys":         storage.Gauge(m.GCSys),
		"HeapAlloc":     storage.Gauge(m.HeapAlloc),
		"HeapIdle":      storage.Gauge(m.HeapIdle),
		"HeapInuse":     storage.Gauge(m.HeapInuse),
		"HeapObjects":   storage.Gauge(m.HeapObjects),
		"HeapReleased":  storage.Gauge(m.HeapReleased),
		"HeapSys":       storage.Gauge(m.HeapSys),
		"LastGC":        storage.Gauge(m.LastGC),
		"Lookups":       storage.Gauge(m.Lookups),
		"MCacheInuse":   storage.Gauge(m.MCacheInuse),
		"MCacheSys":     storage.Gauge(m.MCacheSys),
		"MSpanInuse":    storage.Gauge(m.MSpanInuse),
		"MSpanSys":      storage.Gauge(m.MSpanSys),
		"Mallocs":       storage.Gauge(m.Mallocs),
		"NextGC":        storage.Gauge(m.NextGC),
		"NumForcedGC":   storage.Gauge(m.NumForcedGC),
		"NumGC":         storage.Gauge(m.NumGC),
		"OtherSys":      storage.Gauge(m.OtherSys),
		"PauseTotalNs":  storage.Gauge(m.PauseTotalNs),
		"StackInuse":    storage.Gauge(m.StackInuse),
		"StackSys":      storage.Gauge(m.StackSys),
		"Sys":           storage.Gauge(m.Sys),
		"TotalAlloc":    storage.Gauge(m.TotalAlloc),
		"RandomValue":   storage.Gauge(rand.Float64()),
	}

	counters := map[string]storage.Counter{
		"PollCount": storage.Counter(1),
	}

	if err := a.storage.SetGauges(ctx, gauges); err != nil {
		a.logger.Error("can't set gauges", zap.Error(err))
	}

	if err := a.storage.SetCounters(ctx, counters); err != nil {
		a.logger.Error("can't set counters", zap.Error(err))
	}
}

func (a *Agent) pushMetrics() {
	ctx := context.Background()

	gauges, err := a.storage.GetGauges(ctx)
	if err != nil {
		a.logger.Error("cant get gauges", zap.Error(err))
	}

	counters, err := a.storage.GetCounters(ctx)
	if err != nil {
		a.logger.Error("cant get counters", zap.Error(err))
	}

	metrics := make([]Metric, 0, len(gauges)+len(counters))

	for name, value := range gauges {
		val := float64(value)
		metric := Metric{
			ID:    name,
			MType: "gauge",
			Value: &val,
		}
		metrics = append(metrics, metric)
		if err := a.storage.ClearGauge(ctx, name); err != nil {
			a.logger.Error("cant clear gauges", zap.Error(err))
		}
	}

	for name, value := range counters {
		delta := int64(value)
		metric := Metric{
			ID:    name,
			MType: "counter",
			Delta: &delta,
		}
		metrics = append(metrics, metric)
		if err := a.storage.ClearCounter(ctx, name); err != nil {
			a.logger.Error("cant clear counters", zap.Error(err))
		}
	}

	if len(metrics) > 0 {
		a.pushMetricsBatch(metrics)
	}
}

func (a *Agent) pushMetricsBatch(metrics []Metric) {
	var compressed bytes.Buffer
	writer := gzip.NewWriter(&compressed)
	if err := json.NewEncoder(writer).Encode(metrics); err != nil {
		a.logger.Error("cant gzip metrics", zap.Error(err))
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
		Post(a.serverURL + "/updates/")

	if err != nil {
		a.logger.Error("cant send metrics", zap.Error(err))
		return
	}
	if resp.StatusCode() != http.StatusOK {
		a.logger.Error("bad status code", zap.Error(err))
		return
	}
	a.logger.Info("push success")
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
