package agent

import (
	"bytes"
	"compress/gzip"
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
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	a.storage.SetGauge("Alloc", storage.Gauge(m.Alloc))
	a.storage.SetGauge("BuckHashSys", storage.Gauge(m.BuckHashSys))
	a.storage.SetGauge("Frees", storage.Gauge(m.Frees))
	a.storage.SetGauge("GCCPUFraction", storage.Gauge(m.GCCPUFraction))
	a.storage.SetGauge("GCSys", storage.Gauge(m.GCSys))
	a.storage.SetGauge("HeapAlloc", storage.Gauge(m.HeapAlloc))
	a.storage.SetGauge("HeapIdle", storage.Gauge(m.HeapIdle))
	a.storage.SetGauge("HeapInuse", storage.Gauge(m.HeapInuse))
	a.storage.SetGauge("HeapObjects", storage.Gauge(m.HeapObjects))
	a.storage.SetGauge("HeapReleased", storage.Gauge(m.HeapReleased))
	a.storage.SetGauge("HeapSys", storage.Gauge(m.HeapSys))
	a.storage.SetGauge("LastGC", storage.Gauge(m.LastGC))
	a.storage.SetGauge("Lookups", storage.Gauge(m.Lookups))
	a.storage.SetGauge("MCacheInuse", storage.Gauge(m.MCacheInuse))
	a.storage.SetGauge("MCacheSys", storage.Gauge(m.MCacheSys))
	a.storage.SetGauge("MSpanInuse", storage.Gauge(m.MSpanInuse))
	a.storage.SetGauge("MSpanSys", storage.Gauge(m.MSpanSys))
	a.storage.SetGauge("Mallocs", storage.Gauge(m.Mallocs))
	a.storage.SetGauge("NextGC", storage.Gauge(m.NextGC))
	a.storage.SetGauge("NumForcedGC", storage.Gauge(m.NumForcedGC))
	a.storage.SetGauge("NumGC", storage.Gauge(m.NumGC))
	a.storage.SetGauge("OtherSys", storage.Gauge(m.OtherSys))
	a.storage.SetGauge("PauseTotalNs", storage.Gauge(m.PauseTotalNs))
	a.storage.SetGauge("StackInuse", storage.Gauge(m.StackInuse))
	a.storage.SetGauge("StackSys", storage.Gauge(m.StackSys))
	a.storage.SetGauge("Sys", storage.Gauge(m.Sys))
	a.storage.SetGauge("TotalAlloc", storage.Gauge(m.TotalAlloc))

	a.storage.SetCounter("PollCount", storage.Counter(1))
	a.storage.SetGauge("RandomValue", storage.Gauge(rand.Float64()))
}

func (a *Agent) pushMetrics() {
	for name, value := range a.storage.GetAllGauges() {
		val := float64(value)
		metric := Metrics{
			ID:    name,
			MType: "gauge",
			Value: &val,
		}
		a.pushMetric(metric)
		a.storage.ClearGauge(name)
	}

	for name, value := range a.storage.GetAllCounters() {
		delta := int64(value)
		metric := Metrics{
			ID:    name,
			MType: "counter",
			Delta: &delta,
		}
		a.pushMetric(metric)
		a.storage.ClearCounter(name)
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
		a.logger.Error("cant ping db", zap.Error(err))
		return
	}
	if resp.StatusCode() != http.StatusOK {
		a.logger.Error("ping db fail", zap.Error(err))
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
