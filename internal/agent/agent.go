package agent

import (
	"fmt"
	"log"
	"math/rand/v2"
	"net/http"
	"runtime"
	"time"

	"github.com/go-resty/resty/v2"

	"metrics/internal/storage"
)

type Agent struct {
	storage      storage.MetricsStorage
	client       *resty.Client
	serverURL    string
	pollInterval time.Duration
	pushInterval time.Duration
}

func NewAgent(
	serverURL string,
	metricsStorage storage.MetricsStorage,
	pollInterval, pushInterval time.Duration) *Agent {
	return &Agent{
		serverURL:    serverURL,
		pollInterval: pollInterval,
		pushInterval: pushInterval,
		storage:      metricsStorage,
		client:       resty.New(),
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

func (a *Agent) pushMetrics() error {
	for name, value := range a.storage.GetAllGauges() {
		url := fmt.Sprintf("%s/update/gauge/%s/%v", a.serverURL, name, value)
		if err := a.pushMetric(url); err != nil {
			return err
		}
		a.storage.ClearGauge(name)
	}

	for name, value := range a.storage.GetAllCounters() {
		url := fmt.Sprintf("%s/update/counter/%s/%v", a.serverURL, name, value)
		if err := a.pushMetric(url); err != nil {
			return err
		}
		a.storage.ClearCounter(name)
	}

	return nil
}

func (a *Agent) pushMetric(url string) error {
	resp, err := a.client.R().
		SetHeader("Content-Type", "text/plain").
		Post(url)

	if err != nil {
		return fmt.Errorf("cant send metric: %w", err)
	}
	if resp.StatusCode() != http.StatusOK {
		return fmt.Errorf("bad status code: %d url %s", resp.StatusCode(), url)
	}
	return nil
}

func (a *Agent) Start() error {
	pollTicker := time.NewTicker(a.pollInterval)
	reportTicker := time.NewTicker(a.pushInterval)

	for {
		select {
		case <-pollTicker.C:
			a.pollMetrics()
		case <-reportTicker.C:
			if err := a.pushMetrics(); err != nil {
				log.Printf("Cant push metric error: %v", err)
			}
		}
	}
}
