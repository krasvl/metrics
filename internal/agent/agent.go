package agent

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"math/rand/v2"
	"net/http"
	"runtime"
	"sync"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/mem"
	"go.uber.org/zap"

	"metrics/internal/storage"
)

type Agent struct {
	storage      storage.MetricsStorage
	client       *resty.Client
	logger       *zap.Logger
	serverURL    string
	key          string
	pollInterval time.Duration
	pushInterval time.Duration
	rateLimit    int
	storageMutex sync.RWMutex
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
	key string,
	rateLimit int,
	logger *zap.Logger) *Agent {
	return &Agent{
		serverURL:    serverURL,
		pollInterval: pollInterval,
		pushInterval: pushInterval,
		key:          key,
		rateLimit:    rateLimit,
		storage:      metricsStorage,
		client:       resty.New(),
		logger:       logger,
	}
}

func (a *Agent) pollDefaultMetrics() {
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

	a.storageMutex.Lock()
	if err := a.storage.SetGauges(ctx, gauges); err != nil {
		a.logger.Error("can't set gauges", zap.Error(err))
		return
	}
	if err := a.storage.SetCounters(ctx, counters); err != nil {
		a.logger.Error("can't set counters", zap.Error(err))
		return
	}
	a.storageMutex.Unlock()

	a.logger.Info("poll Default success")
}

func (a *Agent) pollGopsutilMetrics() {
	ctx := context.Background()
	v, err := mem.VirtualMemory()
	if err != nil {
		a.logger.Error("can't get memory stats", zap.Error(err))
		return
	}
	cpuUsage, err := cpu.Percent(0, false)
	if err != nil {
		a.logger.Error("can't get CPU stats", zap.Error(err))
		return
	}

	gauges := map[string]storage.Gauge{
		"TotalMemory":     storage.Gauge(v.Total),
		"FreeMemory":      storage.Gauge(v.Free),
		"CPUutilization1": storage.Gauge(cpuUsage[0]),
	}

	a.storageMutex.Lock()
	if err := a.storage.SetGauges(ctx, gauges); err != nil {
		a.logger.Error("can't set gauges", zap.Error(err))
		return
	}
	a.storageMutex.Unlock()

	a.logger.Info("poll Gopsutil success")
}

func (a *Agent) getMetrics() []Metric {
	ctx := context.Background()

	a.storageMutex.RLock()
	gauges, err := a.storage.GetGauges(ctx)
	if err != nil {
		a.logger.Error("cant get gauges", zap.Error(err))
	}
	counters, err := a.storage.GetCounters(ctx)
	if err != nil {
		a.logger.Error("cant get counters", zap.Error(err))
	}
	a.storageMutex.RUnlock()

	metrics := make([]Metric, 0, len(gauges)+len(counters))

	for name, value := range gauges {
		val := float64(value)
		metric := Metric{
			ID:    name,
			MType: "gauge",
			Value: &val,
		}
		metrics = append(metrics, metric)

		a.storageMutex.Lock()
		if err := a.storage.ClearGauge(ctx, name); err != nil {
			a.logger.Error("cant clear gauges", zap.Error(err))
		}
		a.storageMutex.Unlock()
	}

	for name, value := range counters {
		delta := int64(value)
		metric := Metric{
			ID:    name,
			MType: "counter",
			Delta: &delta,
		}
		metrics = append(metrics, metric)

		a.storageMutex.Lock()
		if err := a.storage.ClearCounter(ctx, name); err != nil {
			a.logger.Error("cant clear counters", zap.Error(err))
		}
		a.storageMutex.Unlock()
	}

	return metrics
}

func (a *Agent) pushMetrics(metrics []Metric) {
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

	hash := a.getHash(compressed.Bytes())

	resp, err := a.withRetry(func() (*resty.Response, error) {
		return a.client.R().
			SetHeader("Content-Type", "application/json").
			SetHeader("Content-Encoding", "gzip").
			SetHeader("HashSHA256", hash).
			SetBody(compressed.Bytes()).
			Post(a.serverURL + "/updates/")
	})

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
	resp, err := a.withRetry(func() (*resty.Response, error) {
		return a.client.R().Get(a.serverURL + "/ping")
	})

	if err != nil {
		a.logger.Warn("cant ping db")
		return
	}
	if resp.StatusCode() != http.StatusOK {
		a.logger.Warn("ping db fail")
		return
	}
	a.logger.Info("ping success")
}

func (a *Agent) withRetry(request func() (*resty.Response, error)) (*resty.Response, error) {
	var resp *resty.Response
	var err error
	for _, delay := range []int{0, 1, 3, 5} {
		time.Sleep(time.Duration(delay) * time.Second)
		resp, err = request()
		if resp.StatusCode() != http.StatusServiceUnavailable {
			return resp, err
		}
		a.logger.Warn("service unavailable, retry", zap.Error(err))
	}
	return resp, err
}

func (a *Agent) getHash(data []byte) string {
	if a.key == "" {
		return ""
	}
	h := hmac.New(sha256.New, []byte(a.key))
	h.Write(data)
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

func (a *Agent) pushWorker(jobs <-chan []Metric) {
	for metrics := range jobs {
		a.pushMetrics(metrics)
	}
}

func (a *Agent) Start() error {
	pollTicker := time.NewTicker(a.pollInterval)
	reportTicker := time.NewTicker(a.pushInterval)

	a.testPing()

	jobs := make(chan []Metric, a.rateLimit)
	for range a.rateLimit {
		go a.pushWorker(jobs)
	}

	for {
		select {
		case <-pollTicker.C:
			go a.pollDefaultMetrics()
			go a.pollGopsutilMetrics()
		case <-reportTicker.C:
			select {
			case jobs <- a.getMetrics():
			default:
				a.logger.Warn("cant push metrics, workers busy")
			}
		}
	}
}
