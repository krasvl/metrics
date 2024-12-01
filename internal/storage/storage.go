package storage

type Gauge float64
type Counter int

type MetricsStorage interface {
	GetAllGauges() []string
	GetGauge(name string) (Gauge, bool)
	SetGauge(name string, value Gauge)

	GetAllCounters() []string
	GetCounter(name string) (Counter, bool)
	SetCounter(name string, value Counter)
}
