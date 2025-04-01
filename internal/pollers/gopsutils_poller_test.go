package pollers

import (
	"context"
	"metrics/internal/storage"
	"testing"
)

func TestGopsutilPoller_Poll(t *testing.T) {
	memStorage := storage.NewMemStorage()
	poller := NewGopsutilPoller(memStorage)

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
}

func TestGopsutilPoller_GetMetrics(t *testing.T) {
	memStorage := storage.NewMemStorage()
	poller := NewGopsutilPoller(memStorage)

	gauges := map[string]storage.Gauge{
		"TotalMemory": 1024,
		"FreeMemory":  512,
	}
	if err := memStorage.SetGauges(context.Background(), gauges); err != nil {
		t.Fatalf("Failed to set gauges: %v", err)
	}
	if err := memStorage.SetCounters(context.Background(), map[string]storage.Counter{"PollCount": 1}); err != nil {
		t.Fatalf("Failed to set counters: %v", err)
	}

	metrics, err := poller.GetMetrics(context.Background())
	if err != nil {
		t.Fatalf("unexpected error during GetMetrics: %v", err)
	}

	if len(metrics) != 3 {
		t.Errorf("expected 3 metrics, got %d", len(metrics))
	}
}

func TestGopsutilPoller_ResetMetrics(t *testing.T) {
	memStorage := storage.NewMemStorage()
	poller := NewGopsutilPoller(memStorage)

	gauges := map[string]storage.Gauge{
		"TotalMemory": 1024,
		"FreeMemory":  512,
	}
	if err := memStorage.SetGauges(context.Background(), gauges); err != nil {
		t.Fatalf("Failed to set gauges: %v", err)
	}
	if err := memStorage.SetCounters(context.Background(), map[string]storage.Counter{"PollCount": 1}); err != nil {
		t.Fatalf("Failed to set counters: %v", err)
	}

	err := poller.ResetMetrics(context.Background())
	if err != nil {
		t.Fatalf("unexpected error during ResetMetrics: %v", err)
	}

	gauges, err = memStorage.GetGauges(context.Background())
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

func TestGopsutilPoller_CollectsAllMetrics(t *testing.T) {
	memStorage := storage.NewMemStorage()
	poller := NewGopsutilPoller(memStorage)

	err := poller.Poll()
	if err != nil {
		t.Fatalf("unexpected error during Poll: %v", err)
	}

	gauges, err := memStorage.GetGauges(context.Background())
	if err != nil {
		t.Fatalf("unexpected error getting gauges: %v", err)
	}

	expectedMetrics := []string{"TotalMemory", "FreeMemory", "CPUutilization1"}
	for _, metric := range expectedMetrics {
		if _, exists := gauges[metric]; !exists {
			t.Errorf("expected metric %s to be collected, but it was not found", metric)
		}
	}
}
