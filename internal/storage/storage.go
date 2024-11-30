package storage

type Gauge float32
type Counter int32

type MetricsStorage interface {
	GetAllGauges() []string
	GetGauge(name string) (Gauge, bool)
	SetGauge(name string, value Gauge)

	GetAllCounters() []string
	GetCounter(name string) (Counter, bool)
	SetCounter(name string, value Counter)
}
