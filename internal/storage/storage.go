package storage

import (
	"context"
)

type Gauge float64
type Counter int

type MetricsStorage interface {
	GetGauge(ctx context.Context, name string) (Gauge, bool, error)
	GetGauges(ctx context.Context) (map[string]Gauge, error)
	SetGauge(ctx context.Context, name string, value Gauge) error
	SetGauges(ctx context.Context, values map[string]Gauge) error
	ClearGauges(ctx context.Context) error

	GetCounter(ctx context.Context, name string) (Counter, bool, error)
	GetCounters(ctx context.Context) (map[string]Counter, error)
	SetCounter(ctx context.Context, name string, value Counter) error
	SetCounters(ctx context.Context, values map[string]Counter) error
	ClearCounters(ctx context.Context) error
}

type MemStorage struct {
	gauges   map[string]Gauge
	counters map[string]Counter
}

func NewMemStorage() *MemStorage {
	return &MemStorage{
		gauges:   make(map[string]Gauge),
		counters: make(map[string]Counter),
	}
}

func (ms *MemStorage) GetGauges(ctx context.Context) (map[string]Gauge, error) {
	return ms.gauges, nil
}

func (ms *MemStorage) GetGauge(ctx context.Context, name string) (Gauge, bool, error) {
	value, ok := ms.gauges[name]
	if !ok {
		return 0, false, nil
	}
	return value, true, nil
}

func (ms *MemStorage) SetGauge(ctx context.Context, name string, value Gauge) error {
	ms.gauges[name] = value
	return nil
}

func (ms *MemStorage) SetGauges(ctx context.Context, values map[string]Gauge) error {
	for name, value := range values {
		ms.gauges[name] = value
	}
	return nil
}

func (ms *MemStorage) ClearGauges(ctx context.Context) error {
	for k := range ms.gauges {
		delete(ms.gauges, k)
	}
	return nil
}

func (ms *MemStorage) GetCounters(ctx context.Context) (map[string]Counter, error) {
	return ms.counters, nil
}

func (ms *MemStorage) GetCounter(ctx context.Context, name string) (Counter, bool, error) {
	value, ok := ms.counters[name]
	if !ok {
		return 0, false, nil
	}
	return value, true, nil
}

func (ms *MemStorage) SetCounter(ctx context.Context, name string, value Counter) error {
	ms.counters[name] += value
	return nil
}

func (ms *MemStorage) SetCounters(ctx context.Context, values map[string]Counter) error {
	for name, value := range values {
		ms.counters[name] += value
	}
	return nil
}

func (ms *MemStorage) ClearCounters(ctx context.Context) error {
	for k := range ms.counters {
		delete(ms.counters, k)
	}
	return nil
}
