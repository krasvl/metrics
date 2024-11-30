package storage

type Gauge float32
type Counter int32

type MetricsStorage interface {
	GetGauge(name string) Gauge
	SetGauge(name string, value Gauge)

	GetCounter(name string) Counter
	SetCounter(name string, value Counter)
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

func (ms *MemStorage) GetGauge(name string) Gauge {
	return ms.gauges[name]
}

func (ms *MemStorage) SetGauge(name string, value Gauge) {
	ms.gauges[name] = value
}

func (ms *MemStorage) GetCounter(name string) Counter {
	return ms.counters[name]
}

func (ms *MemStorage) SetCounter(name string, value Counter) {
	ms.counters[name] += value
}
