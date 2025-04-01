package pollers

import (
	"context"
	"fmt"
	"math/rand/v2"
	"metrics/internal/storage"
	"runtime"
)

type DefaultPoller struct {
	basePoller
}

func NewDefaultPoller(ms storage.MetricsStorage) *DefaultPoller {
	return &DefaultPoller{basePoller: basePoller{storage: ms}}
}

func (p *DefaultPoller) Poll() error {
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

	if err := p.storeGauges(ctx, gauges); err != nil {
		return fmt.Errorf("can't set gauges: %w", err)
	}
	if err := p.storeCounters(ctx, counters); err != nil {
		return fmt.Errorf("can't set counters: %w", err)
	}

	return nil
}

func (p *DefaultPoller) GetMetrics(ctx context.Context) ([]Metric, error) {
	return p.getMetrics(ctx)
}

func (p *DefaultPoller) ResetMetrics(ctx context.Context) error {
	return p.resetMetrics(ctx)
}
