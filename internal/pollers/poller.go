package pollers

import (
	"context"
	"fmt"
	"metrics/internal/storage"
	"sync"
)

// MType represents the type of a metric (gauge or counter).
type MType string

const (
	// TypeGauge represents a gauge metric type.
	TypeGauge MType = "gauge"
	// TypeCounter represents a counter metric type.
	TypeCounter MType = "counter"
)

// Metric represents a single metric with its type and value.
type Metric struct {
	Delta *int64   `json:"delta,omitempty"`
	Value *float64 `json:"value,omitempty"`
	ID    string   `json:"id"`
	MType MType    `json:"type"`
}

// Poller defines the interface for metric pollers.
type Poller interface {
	// Poll collects metrics and stores them in the storage.
	Poll() error
	// GetMetrics retrieves all collected metrics from the storage.
	GetMetrics(ctx context.Context) ([]Metric, error)
	// ResetMetrics clears all collected metrics from the storage.
	ResetMetrics(ctx context.Context) error
}

type basePoller struct {
	storage storage.MetricsStorage
	mu      sync.RWMutex
}

func (b *basePoller) storeGauges(ctx context.Context, gauges map[string]storage.Gauge) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.storage.SetGauges(ctx, gauges)
}

func (b *basePoller) storeCounters(ctx context.Context, counters map[string]storage.Counter) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.storage.SetCounters(ctx, counters)
}

func (b *basePoller) getMetrics(ctx context.Context) ([]Metric, error) {
	b.mu.RLock()
	gauges, err := b.storage.GetGauges(ctx)
	if err != nil {
		b.mu.RUnlock()
		return nil, fmt.Errorf("can't get gauges: %w", err)
	}
	counters, err := b.storage.GetCounters(ctx)
	if err != nil {
		b.mu.RUnlock()
		return nil, fmt.Errorf("can't get counters: %w", err)
	}
	b.mu.RUnlock()

	metrics := make([]Metric, 0, len(gauges)+len(counters))
	for name, value := range gauges {
		v := float64(value)
		metrics = append(metrics, Metric{
			Value: &v,
			ID:    name,
			MType: TypeGauge,
		})
	}
	for name, value := range counters {
		d := int64(value)
		metrics = append(metrics, Metric{
			Delta: &d,
			ID:    name,
			MType: TypeCounter,
		})
	}

	return metrics, nil
}

func (b *basePoller) resetMetrics(ctx context.Context) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := b.storage.ClearGauges(ctx); err != nil {
		return fmt.Errorf("can't clear gauges: %w", err)
	}
	if err := b.storage.ClearCounters(ctx); err != nil {
		return fmt.Errorf("can't clear counters: %w", err)
	}
	return nil
}
