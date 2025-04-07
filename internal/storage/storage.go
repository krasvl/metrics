package storage

import (
	"context"
)

// Gauge represents a floating-point metric value.
type Gauge float64

// Counter represents an integer metric value.
type Counter int

// MetricsStorage defines the interface for metric storage operations.
type MetricsStorage interface {
	// GetGauge retrieves a gauge metric by name.
	GetGauge(ctx context.Context, name string) (Gauge, bool, error)
	// GetGauges retrieves all gauge metrics.
	GetGauges(ctx context.Context) (map[string]Gauge, error)
	// SetGauge sets a gauge metric by name and value.
	SetGauge(ctx context.Context, name string, value Gauge) error
	// SetGauges sets multiple gauge metrics.
	SetGauges(ctx context.Context, values map[string]Gauge) error
	// ClearGauges clears all gauge metrics.
	ClearGauges(ctx context.Context) error

	// GetCounter retrieves a counter metric by name.
	GetCounter(ctx context.Context, name string) (Counter, bool, error)
	// GetCounters retrieves all counter metrics.
	GetCounters(ctx context.Context) (map[string]Counter, error)
	// SetCounter increments a counter metric by name and value.
	SetCounter(ctx context.Context, name string, value Counter) error
	// SetCounters increments multiple counter metrics.
	SetCounters(ctx context.Context, values map[string]Counter) error
	// ClearCounters clears all counter metrics.
	ClearCounters(ctx context.Context) error
}

// MemStorage is an in-memory implementation of MetricsStorage.
type MemStorage struct {
	gauges   map[string]Gauge
	counters map[string]Counter
}

// NewMemStorage creates a new instance of MemStorage.
func NewMemStorage() *MemStorage {
	return &MemStorage{
		gauges:   make(map[string]Gauge),
		counters: make(map[string]Counter),
	}
}

// GetGauges retrieves all gauge metrics from memory.
func (ms *MemStorage) GetGauges(ctx context.Context) (map[string]Gauge, error) {
	return ms.gauges, nil
}

// GetGauge retrieves a specific gauge metric by name from memory.
func (ms *MemStorage) GetGauge(ctx context.Context, name string) (Gauge, bool, error) {
	value, ok := ms.gauges[name]
	if !ok {
		return 0, false, nil
	}
	return value, true, nil
}

// SetGauge sets a gauge metric in memory.
func (ms *MemStorage) SetGauge(ctx context.Context, name string, value Gauge) error {
	ms.gauges[name] = value
	return nil
}

// SetGauges sets multiple gauge metrics in memory.
func (ms *MemStorage) SetGauges(ctx context.Context, values map[string]Gauge) error {
	for name, value := range values {
		ms.gauges[name] = value
	}
	return nil
}

// ClearGauges clears all gauge metrics from memory.
func (ms *MemStorage) ClearGauges(ctx context.Context) error {
	for k := range ms.gauges {
		delete(ms.gauges, k)
	}
	return nil
}

// GetCounters retrieves all counter metrics from memory.
func (ms *MemStorage) GetCounters(ctx context.Context) (map[string]Counter, error) {
	return ms.counters, nil
}

// GetCounter retrieves a specific counter metric by name from memory.
func (ms *MemStorage) GetCounter(ctx context.Context, name string) (Counter, bool, error) {
	value, ok := ms.counters[name]
	if !ok {
		return 0, false, nil
	}
	return value, true, nil
}

// SetCounter increments a counter metric in memory.
func (ms *MemStorage) SetCounter(ctx context.Context, name string, value Counter) error {
	ms.counters[name] += value
	return nil
}

// SetCounters increments multiple counter metrics in memory.
func (ms *MemStorage) SetCounters(ctx context.Context, values map[string]Counter) error {
	for name, value := range values {
		ms.counters[name] += value
	}
	return nil
}

// ClearCounters clears all counter metrics from memory.
func (ms *MemStorage) ClearCounters(ctx context.Context) error {
	for k := range ms.counters {
		delete(ms.counters, k)
	}
	return nil
}
