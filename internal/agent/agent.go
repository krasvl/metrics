package agent

import (
	"fmt"
	"math/rand/v2"
	"net/http"
	"runtime"
	"time"

	"metrics/internal/storage"
)

type Agent struct {
	serverUrl    string
	pollInterval time.Duration
	pushInterval time.Duration

	storage storage.MetricsStorage
}

func NewAgent(serverUrl string, storage storage.MetricsStorage, pollInterval, pushInterval time.Duration) *Agent {
	return &Agent{
		serverUrl:    serverUrl,
		pollInterval: pollInterval,
		pushInterval: pushInterval,

		storage: storage,
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

	gauges := a.storage.GetAllGauges()
	for _, name := range gauges {
		value, _ := a.storage.GetGauge(name)
		url := fmt.Sprintf("%s/update/gauge/%s/%v", a.serverUrl, name, value)
		pushMetric(url)
	}

	counters := a.storage.GetAllCounters()
	for _, name := range counters {
		value, _ := a.storage.GetCounter(name)
		url := fmt.Sprintf("%s/update/counter/%s/%v", a.serverUrl, name, value)
		pushMetric(url)
	}
	return nil
}

func pushMetric(url string) error {
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return fmt.Errorf("cant create request: %w", err)
	}
	req.Header.Set("Content-Type", "text/plain")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("cant send metric: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status code: %d url %s", resp.StatusCode, url)
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
			a.pushMetrics()
		}
	}
}
