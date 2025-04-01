package storage

import (
	"context"
	"testing"
)

func TestMemStorage_SetAndGetGauges(t *testing.T) {
	memStorage := NewMemStorage()

	testCases := []struct {
		name     string
		values   map[string]Gauge
		expected map[string]Gauge
	}{
		{
			name: "single gauge",
			values: map[string]Gauge{
				"gauge1": 1.0,
			},
			expected: map[string]Gauge{
				"gauge1": 1.0,
			},
		},
		{
			name: "multiple gauges",
			values: map[string]Gauge{
				"gauge1": 1.0,
				"gauge2": 2.0,
			},
			expected: map[string]Gauge{
				"gauge1": 1.0,
				"gauge2": 2.0,
			},
		},
		{
			name: "large values",
			values: map[string]Gauge{
				"gauge1": 1e10,
				"gauge2": 2e10,
			},
			expected: map[string]Gauge{
				"gauge1": 1e10,
				"gauge2": 2e10,
			},
		},
		{
			name: "small values",
			values: map[string]Gauge{
				"gauge1": 1e-10,
				"gauge2": 2e-10,
			},
			expected: map[string]Gauge{
				"gauge1": 1e-10,
				"gauge2": 2e-10,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := memStorage.SetGauges(context.Background(), tc.values)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			gauges, err := memStorage.GetGauges(context.Background())
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			for k, v := range tc.expected {
				if gauges[k] != v {
					t.Errorf("expected %s to be %f, got %f", k, v, gauges[k])
				}
			}
		})
	}
}

func TestMemStorage_SetAndGetCounters(t *testing.T) {
	memStorage := NewMemStorage()

	testCases := []struct {
		name     string
		values   map[string]Counter
		expected map[string]Counter
	}{
		{
			name: "single counter",
			values: map[string]Counter{
				"counter1": 1,
			},
			expected: map[string]Counter{
				"counter1": 1,
			},
		},
		{
			name: "multiple counters",
			values: map[string]Counter{
				"counter1": 1,
				"counter2": 2,
			},
			expected: map[string]Counter{
				"counter1": 1 + 1,
				"counter2": 2,
			},
		},
		{
			name: "large values",
			values: map[string]Counter{
				"counter1": 1e10,
				"counter2": 2e10,
			},
			expected: map[string]Counter{
				"counter1": 1 + 1 + 1e10,
				"counter2": 2 + 2e10,
			},
		},
		{
			name: "small values",
			values: map[string]Counter{
				"counter1": 1,
				"counter2": 2,
			},
			expected: map[string]Counter{
				"counter1": 1 + 1 + 1e10 + 1,
				"counter2": 2 + 2e10 + 2,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := memStorage.SetCounters(context.Background(), tc.values)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			counters, err := memStorage.GetCounters(context.Background())
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			for k, v := range tc.expected {
				if counters[k] != v {
					t.Errorf("expected %s to be %d, got %d", k, v, counters[k])
				}
			}
		})
	}
}

func TestMemStorage_ClearGauges(t *testing.T) {
	memStorage := NewMemStorage()

	initialGauges := map[string]Gauge{"gauge1": 1.0, "gauge2": 2.0}
	err := memStorage.SetGauges(context.Background(), initialGauges)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = memStorage.ClearGauges(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	gauges, err := memStorage.GetGauges(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(gauges) != 0 {
		t.Errorf("expected no gauges, got %v", gauges)
	}
}

func TestMemStorage_ClearCounters(t *testing.T) {
	memStorage := NewMemStorage()

	initialCounters := map[string]Counter{"counter1": 10, "counter2": 20}
	err := memStorage.SetCounters(context.Background(), initialCounters)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = memStorage.ClearCounters(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	counters, err := memStorage.GetCounters(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(counters) != 0 {
		t.Errorf("expected no counters, got %v", counters)
	}
}
