package storage

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

func (ms *MemStorage) GetAllGauges() []string {
	names := make([]string, 0, len(ms.gauges))

	for name := range ms.gauges {
		names = append(names, name)
	}

	return names
}

func (ms *MemStorage) GetGauge(name string) (Gauge, bool) {
	value, exist := ms.gauges[name]
	return value, exist
}

func (ms *MemStorage) SetGauge(name string, value Gauge) {
	ms.gauges[name] = value
}

func (ms *MemStorage) GetAllCounters() []string {
	names := make([]string, 0, len(ms.counters))

	for name := range ms.counters {
		names = append(names, name)
	}

	return names
}

func (ms *MemStorage) GetCounter(name string) (Counter, bool) {
	value, exist := ms.counters[name]
	return value, exist
}

func (ms *MemStorage) SetCounter(name string, value Counter) {
	ms.counters[name] += value
}
