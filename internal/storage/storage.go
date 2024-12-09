package storage

type Gauge float64
type Counter int

type MetricsStorage interface {
	GetAllGauges() map[string]Gauge
	GetGauge(name string) (Gauge, bool)
	SetGauge(name string, value Gauge)
	ClearGauge(name string)
	ClearGauges()

	GetAllCounters() map[string]Counter
	GetCounter(name string) (Counter, bool)
	SetCounter(name string, value Counter)
	ClearCounter(name string)
	ClearCounters()
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

func (ms *MemStorage) GetAllGauges() map[string]Gauge {
	return ms.gauges
}

func (ms *MemStorage) GetGauge(name string) (Gauge, bool) {
	value, ok := ms.gauges[name]
	return value, ok
}

func (ms *MemStorage) SetGauge(name string, value Gauge) {
	ms.gauges[name] = value
}

func (ms *MemStorage) ClearGauge(name string) {
	delete(ms.gauges, name)
}

func (ms *MemStorage) ClearGauges() {
	for k := range ms.gauges {
		delete(ms.gauges, k)
	}
}

func (ms *MemStorage) GetAllCounters() map[string]Counter {
	return ms.counters
}

func (ms *MemStorage) GetCounter(name string) (Counter, bool) {
	value, ok := ms.counters[name]
	return value, ok
}

func (ms *MemStorage) SetCounter(name string, value Counter) {
	ms.counters[name] += value
}

func (ms *MemStorage) ClearCounter(name string) {
	delete(ms.counters, name)
}

func (ms *MemStorage) ClearCounters() {
	for k := range ms.counters {
		delete(ms.counters, k)
	}
}
