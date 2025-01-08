package agent

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"metrics/internal/storage"
)

func TestPushMetrics(t *testing.T) {
	memStorage := storage.NewMemStorage()
	agent := NewAgent("http://localhost:8080", memStorage, 2*time.Second, 10*time.Second)

	agent.storage.SetGauge("testGauge1", storage.Gauge(1.1))
	agent.storage.SetGauge("testGauge2", storage.Gauge(1.2))
	agent.storage.SetCounter("testCounter1", storage.Counter(1))
	agent.storage.SetCounter("testCounter2", storage.Counter(2))

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		if r.Header.Get("Content-Type") != "application/json" {
			http.Error(w, "Bad Content-Type", http.StatusUnsupportedMediaType)
			return
		}

		var metric Metrics
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

	if err := agent.pushMetrics(); err != nil {
		t.Errorf("expected no error, got %v", err)
		return
	}
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

	memStorage := storage.NewMemStorage()
	agent := NewAgent("http://localhost:8080", memStorage, 2*time.Second, 10*time.Second)

	agent.pollMetrics()

	for _, name := range expectedGauges {
		if _, ok := agent.storage.GetGauge(name); !ok {
			t.Errorf("expected gauge metric %s to be collected", name)
		}
	}

	for _, name := range expectedConters {
		if _, ok := agent.storage.GetCounter(name); !ok {
			t.Errorf("expected counter metric %s to be collected", name)
		}
	}
}

func TestPollCounter(t *testing.T) {
	memStorage := storage.NewMemStorage()
	agent := NewAgent("http://localhost:8080", memStorage, 2*time.Second, 10*time.Second)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer ts.Close()

	agent.serverURL = ts.URL

	agent.pollMetrics()
	agent.pollMetrics()
	if v, _ := agent.storage.GetCounter("PollCount"); v != 2 {
		t.Errorf("expected PollCount 2, got %d", v)
		return
	}

	if err := agent.pushMetrics(); err != nil {
		t.Errorf("expected no error, got %v", err)
		return
	}

	if v, _ := agent.storage.GetCounter("PollCount"); v != 0 {
		t.Errorf("expected PollCount 0, got %d", v)
		return
	}

	agent.pollMetrics()
	if v, _ := agent.storage.GetCounter("PollCount"); v != 1 {
		t.Errorf("expected PollCount 1, got %d", v)
		return
	}
}
