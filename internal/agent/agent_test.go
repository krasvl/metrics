package agent

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"metrics/internal/storage"

	"go.uber.org/zap"
)

func TestPushMetrics(t *testing.T) {
	memStorage := storage.NewMemStorage()
	logger, _ := zap.NewProduction()
	ctx := context.Background()
	agent := NewAgent("http://localhost:8080", memStorage, 2*time.Second, 10*time.Second, "", 1000, logger)

	_ = agent.storage.SetGauge(ctx, "testGauge1", storage.Gauge(1.1))
	_ = agent.storage.SetGauge(ctx, "testGauge2", storage.Gauge(1.2))
	_ = agent.storage.SetCounter(ctx, "testCounter1", storage.Counter(1))
	_ = agent.storage.SetCounter(ctx, "testCounter2", storage.Counter(2))

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		if r.Header.Get("Content-Type") != "application/json" {
			http.Error(w, "Bad Content-Type", http.StatusUnsupportedMediaType)
			return
		}

		if r.Header.Get("Content-Encoding") != "gzip" {
			http.Error(w, "Bad Content-Encoding", http.StatusUnsupportedMediaType)
			return
		}

		gz, err := gzip.NewReader(r.Body)
		if err != nil {
			http.Error(w, "cant gzip body", http.StatusInternalServerError)
			return
		}
		defer func() {
			if err := gz.Close(); err != nil {
				http.Error(w, "cant close gzip writer", http.StatusInternalServerError)
			}
		}()
		r.Body = gz

		var metric Metric
		if err := json.NewDecoder(r.Body).Decode(&metric); err != nil {
			http.Error(w, "Bad JSON", http.StatusBadRequest)
			return
		}

		switch metric.ID {
		case "testGauge1":
			if metric.MType != "gauge" || *metric.Value != 1.1 {
				http.Error(w, "Invalid metric", http.StatusBadRequest)
				return
			}
		case "testGauge2":
			if metric.MType != "gauge" || *metric.Value != 1.2 {
				http.Error(w, "Invalid metric", http.StatusBadRequest)
				return
			}
		case "testCounter1":
			if metric.MType != "counter" || *metric.Delta != 1 {
				http.Error(w, "Invalid metric", http.StatusBadRequest)
				return
			}
		case "testCounter2":
			if metric.MType != "counter" || *metric.Delta != 2 {
				http.Error(w, "Invalid metric", http.StatusBadRequest)
				return
			}
		default:
			http.Error(w, "Not Found", http.StatusNotFound)
			return
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	agent.serverURL = ts.URL

	agent.pushMetrics()
}

func TestPollMetrics(t *testing.T) {
	expectedGauges := []string{
		"Alloc",
		"BuckHashSys",
		"Frees",
		"GCCPUFraction",
		"GCSys",
		"HeapAlloc",
		"HeapIdle",
		"HeapInuse",
		"HeapObjects",
		"HeapReleased",
		"HeapSys",
		"LastGC",
		"Lookups",
		"MCacheInuse",
		"MCacheSys",
		"MSpanInuse",
		"MSpanSys",
		"Mallocs",
		"NextGC",
		"NumForcedGC",
		"NumGC",
		"OtherSys",
		"PauseTotalNs",
		"StackInuse",
		"StackSys",
		"Sys",
		"TotalAlloc",

		"RandomValue",
	}

	expectedConters := []string{
		"PollCount",
	}

	logger, _ := zap.NewProduction()
	memStorage := storage.NewMemStorage()
	ctx := context.Background()
	agent := NewAgent("http://localhost:8080", memStorage, 2*time.Second, 10*time.Second, "", 1000, logger)

	agent.pollDefaultMetrics()

	for _, name := range expectedGauges {
		if _, ok, err := agent.storage.GetGauge(ctx, name); !ok || err != nil {
			t.Errorf("expected gauge metric %s to be collected", name)
		}
	}

	for _, name := range expectedConters {
		if _, ok, err := agent.storage.GetCounter(ctx, name); !ok || err != nil {
			t.Errorf("expected counter metric %s to be collected", name)
		}
	}
}

func TestPollCounter(t *testing.T) {
	logger, _ := zap.NewProduction()
	memStorage := storage.NewMemStorage()
	ctx := context.Background()
	agent := NewAgent("http://localhost:8080", memStorage, 2*time.Second, 10*time.Second, "", 1000, logger)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer ts.Close()

	agent.serverURL = ts.URL

	agent.pollDefaultMetrics()
	agent.pollDefaultMetrics()
	if v, _, _ := agent.storage.GetCounter(ctx, "PollCount"); v != 2 {
		t.Errorf("expected PollCount 2, got %d", v)
		return
	}

	agent.pushMetrics()

	if v, _, _ := agent.storage.GetCounter(ctx, "PollCount"); v != 0 {
		t.Errorf("expected PollCount 0, got %d", v)
		return
	}

	agent.pollDefaultMetrics()
	if v, _, _ := agent.storage.GetCounter(ctx, "PollCount"); v != 1 {
		t.Errorf("expected PollCount 1, got %d", v)
		return
	}
}
