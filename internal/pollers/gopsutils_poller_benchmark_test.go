package pollers

import (
	"context"
	"log"
	"metrics/internal/storage"
	"testing"
)

func BenchmarkGopsutilPoller_Poll(b *testing.B) {
	memStorage := storage.NewMemStorage()
	poller := NewGopsutilPoller(memStorage)

	for i := 0; i < b.N; i++ {
		if err := poller.Poll(); err != nil {
			b.Fatalf("Poll failed: %v", err)
		}
	}
}

func BenchmarkGopsutilPoller_GetMetrics(b *testing.B) {
	ctx := context.Background()
	memStorage := storage.NewMemStorage()
	poller := NewGopsutilPoller(memStorage)

	if err := memStorage.SetGauges(ctx, map[string]storage.Gauge{"TotalMemory": 1024, "FreeMemory": 512}); err != nil {
		log.Println("set gauges error:", err)
	}
	if err := memStorage.SetCounters(ctx, map[string]storage.Counter{"PollCount": 1}); err != nil {
		log.Println("set counters error:", err)
	}

	for i := 0; i < b.N; i++ {
		if _, err := poller.GetMetrics(ctx); err != nil {
			b.Fatalf("GetMetrics failed: %v", err)
		}
	}
}

func BenchmarkGopsutilPoller_ResetMetrics(b *testing.B) {
	ctx := context.Background()
	memStorage := storage.NewMemStorage()
	poller := NewGopsutilPoller(memStorage)

	if err := memStorage.SetGauges(ctx, map[string]storage.Gauge{"TotalMemory": 1024, "FreeMemory": 512}); err != nil {
		log.Println("set gauges error:", err)
	}
	if err := memStorage.SetCounters(ctx, map[string]storage.Counter{"PollCount": 1}); err != nil {
		log.Println("set counters error:", err)
	}

	for i := 0; i < b.N; i++ {
		if err := poller.ResetMetrics(ctx); err != nil {
			b.Fatalf("ResetMetrics failed: %v", err)
		}
	}
}
