package agent

import (
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

		expectedGauge1URL := "/update/gauge/testGauge1/1.1"
		expectedGauge2URL := "/update/gauge/testGauge2/1.2"
		expectedCounter1URL := "/update/counter/testCounter1/1"
		expectedCounter2URL := "/update/counter/testCounter2/2"

		switch r.URL.Path {
		case expectedGauge1URL:
			w.WriteHeader(http.StatusOK)
		case expectedGauge2URL:
			w.WriteHeader(http.StatusOK)
		case expectedCounter1URL:
			w.WriteHeader(http.StatusOK)
		case expectedCounter2URL:
			w.WriteHeader(http.StatusOK)
		default:
			t.Errorf("unexpected URL: got %s", r.URL.Path)
			http.Error(w, "Not Found", http.StatusNotFound)
		}
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
