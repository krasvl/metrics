package pollers

import (
	"context"
	"metrics/internal/storage"
	"testing"
)

func TestDefaultPoller_Poll(t *testing.T) {
	memStorage := storage.NewMemStorage()
	poller := NewDefaultPoller(memStorage)

	err := poller.Poll()
	if err != nil {
		t.Fatalf("unexpected error during Poll: %v", err)
	}

	gauges, err := memStorage.GetGauges(context.Background())
	if err != nil {
		t.Fatalf("unexpected error getting gauges: %v", err)
	}
	if len(gauges) == 0 {
		t.Errorf("expected gauges to be populated, got empty map")
	}

	counters, err := memStorage.GetCounters(context.Background())
	if err != nil {
		t.Fatalf("unexpected error getting counters: %v", err)
	}
	if len(counters) == 0 || counters["PollCount"] != 1 {
		t.Errorf("expected PollCount counter to be 1, got %v", counters["PollCount"])
	}
}

func TestDefaultPoller_GetMetrics(t *testing.T) {
	memStorage := storage.NewMemStorage()
	poller := NewDefaultPoller(memStorage)

	if err := memStorage.SetGauges(context.Background(), map[string]storage.Gauge{"Alloc": 123.45}); err != nil {
		t.Fatalf("Failed to set gauges: %v", err)
	}
	if err := memStorage.SetCounters(context.Background(), map[string]storage.Counter{"PollCount": 5}); err != nil {
		t.Fatalf("Failed to set counters: %v", err)
	}

	metrics, err := poller.GetMetrics(context.Background())
	if err != nil {
		t.Fatalf("unexpected error during GetMetrics: %v", err)
	}

	if len(metrics) != 2 {
		t.Errorf("expected 2 metrics, got %d", len(metrics))
	}
}

func TestDefaultPoller_ResetMetrics(t *testing.T) {
	memStorage := storage.NewMemStorage()
	poller := NewDefaultPoller(memStorage)

	if err := memStorage.SetGauges(context.Background(), map[string]storage.Gauge{"Alloc": 123.45}); err != nil {
		t.Fatalf("Failed to set gauges: %v", err)
	}
	if err := memStorage.SetCounters(context.Background(), map[string]storage.Counter{"PollCount": 5}); err != nil {
		t.Fatalf("Failed to set counters: %v", err)
	}

	err := poller.ResetMetrics(context.Background())
	if err != nil {
		t.Fatalf("unexpected error during ResetMetrics: %v", err)
	}

	gauges, err := memStorage.GetGauges(context.Background())
	if err != nil {
		t.Fatalf("unexpected error getting gauges: %v", err)
	}
	if len(gauges) != 0 {
		t.Errorf("expected no gauges, got %v", gauges)
	}

	counters, err := memStorage.GetCounters(context.Background())
	if err != nil {
		t.Fatalf("unexpected error getting counters: %v", err)
	}
	if len(counters) != 0 {
		t.Errorf("expected no counters, got %v", counters)
	}
}

func TestDefaultPoller_CollectsAllMetrics(t *testing.T) {
	memStorage := storage.NewMemStorage()
	poller := NewDefaultPoller(memStorage)

	err := poller.Poll()
	if err != nil {
		t.Fatalf("unexpected error during Poll: %v", err)
	}

	gauges, err := memStorage.GetGauges(context.Background())
	if err != nil {
		t.Fatalf("unexpected error getting gauges: %v", err)
	}

	expectedMetrics := []string{
		"Alloc", "BuckHashSys", "Frees", "GCCPUFraction", "GCSys", "HeapAlloc",
		"HeapIdle", "HeapInuse", "HeapObjects", "HeapReleased", "HeapSys",
		"LastGC", "Lookups", "MCacheInuse", "MCacheSys", "MSpanInuse",
		"MSpanSys", "Mallocs", "NextGC", "NumForcedGC", "NumGC", "OtherSys",
		"PauseTotalNs", "StackInuse", "StackSys", "Sys", "TotalAlloc", "RandomValue",
	}
	for _, metric := range expectedMetrics {
		if _, exists := gauges[metric]; !exists {
			t.Errorf("expected metric %s to be collected, but it was not found", metric)
		}
	}
}
