package pollers

import (
	"context"
	"metrics/internal/storage"
	"testing"
)

func BenchmarkStoreGauges(b *testing.B) {
	ctx := context.Background()
	mockStorage := storage.NewMemStorage()
	poller := &basePoller{storage: mockStorage}

	gauges := map[string]storage.Gauge{
		"gauge1": 1.23,
		"gauge2": 4.56,
	}

	for i := 0; i < b.N; i++ {
		if err := poller.storeGauges(ctx, gauges); err != nil {
			b.Fatalf("storeGauges failed: %v", err)
		}
	}
}

func BenchmarkStoreCounters(b *testing.B) {
	ctx := context.Background()
	mockStorage := storage.NewMemStorage()
	poller := &basePoller{storage: mockStorage}

	counters := map[string]storage.Counter{
		"counter1": 123,
		"counter2": 456,
	}

	for i := 0; i < b.N; i++ {
		if err := poller.storeCounters(ctx, counters); err != nil {
			b.Fatalf("storeCounters failed: %v", err)
		}
	}
}

func BenchmarkGetMetrics(b *testing.B) {
	ctx := context.Background()
	mockStorage := storage.NewMemStorage()
	poller := &basePoller{storage: mockStorage}

	mockStorage.SetGauges(ctx, map[string]storage.Gauge{"gauge1": 1.23})
	mockStorage.SetCounters(ctx, map[string]storage.Counter{"counter1": 123})

	for i := 0; i < b.N; i++ {
		if _, err := poller.getMetrics(ctx); err != nil {
			b.Fatalf("getMetrics failed: %v", err)
		}
	}
}

func BenchmarkResetMetrics(b *testing.B) {
	ctx := context.Background()
	mockStorage := storage.NewMemStorage()
	poller := &basePoller{storage: mockStorage}

	mockStorage.SetGauges(ctx, map[string]storage.Gauge{"gauge1": 1.23})
	mockStorage.SetCounters(ctx, map[string]storage.Counter{"counter1": 123})

	for i := 0; i < b.N; i++ {
		if err := poller.resetMetrics(ctx); err != nil {
			b.Fatalf("resetMetrics failed: %v", err)
		}
	}
}
