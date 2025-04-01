package pollers

import (
	"context"
	"fmt"
	"metrics/internal/storage"

	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/mem"
)

type GopsutilPoller struct {
	basePoller
}

func NewGopsutilPoller(ms storage.MetricsStorage) *GopsutilPoller {
	return &GopsutilPoller{basePoller: basePoller{storage: ms}}
}

func (p *GopsutilPoller) Poll() error {
	ctx := context.Background()
	v, err := mem.VirtualMemory()
	if err != nil {
		return fmt.Errorf("can't get memory info: %w", err)
	}
	cpuUsage, err := cpu.Percent(0, false)
	if err != nil {
		return fmt.Errorf("can't get CPU info: %w", err)
	}

	gauges := map[string]storage.Gauge{
		"TotalMemory":     storage.Gauge(v.Total),
		"FreeMemory":      storage.Gauge(v.Free),
		"CPUutilization1": storage.Gauge(cpuUsage[0]),
	}

	if err := p.storeGauges(ctx, gauges); err != nil {
		return err
	}

	return nil
}

func (p *GopsutilPoller) GetMetrics(ctx context.Context) ([]Metric, error) {
	return p.getMetrics(ctx)
}

func (p *GopsutilPoller) ResetMetrics(ctx context.Context) error {
	return p.resetMetrics(ctx)
}
