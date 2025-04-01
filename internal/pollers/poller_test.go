package pollers

import (
	"context"
	"metrics/internal/storage"
	"sync"
	"testing"
)

func TestBasePoller_SetGauges(t *testing.T) {
	testCases := []struct {
		name     string
		values   map[string]storage.Gauge
		expected map[string]storage.Gauge
	}{
		{
			name: "single gauge",
			values: map[string]storage.Gauge{
				"gauge1": 1.0,
			},
			expected: map[string]storage.Gauge{
				"gauge1": 1.0,
			},
		},
		{
			name: "multiple gauges",
			values: map[string]storage.Gauge{
				"gauge1": 1.0,
				"gauge2": 2.0,
			},
			expected: map[string]storage.Gauge{
				"gauge1": 1.0,
				"gauge2": 2.0,
			},
		},
		{
			name: "multiple gauges large",
			values: map[string]storage.Gauge{
				"gauge1": 1e10,
				"gauge2": 2e10,
			},
			expected: map[string]storage.Gauge{
				"gauge1": 1e10,
				"gauge2": 2e10,
			},
		},
		{
			name: "multiple gauges small",
			values: map[string]storage.Gauge{
				"gauge1": 1e-10,
				"gauge2": 2e-10,
			},
			expected: map[string]storage.Gauge{
				"gauge1": 1e-10,
				"gauge2": 2e-10,
			},
		},
	}

	memStorage := storage.NewMemStorage()
	poller := basePoller{
		storage: memStorage,
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := poller.storeGauges(context.Background(), tc.values)
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

func TestBasePoller_SetGauges_Concurrent(t *testing.T) {
	testCases := []struct {
		name     string
		values   map[string]storage.Gauge
		expected map[string]storage.Gauge
	}{
		{
			name: "single gauge",
			values: map[string]storage.Gauge{
				"gauge1": 1.0,
			},
			expected: map[string]storage.Gauge{
				"gauge1": 1.0,
			},
		},
		{
			name: "multiple gauges",
			values: map[string]storage.Gauge{
				"gauge1": 1.0,
				"gauge2": 2.0,
			},
			expected: map[string]storage.Gauge{
				"gauge1": 1.0,
				"gauge2": 2.0,
			},
		},
		{
			name: "multiple gauges large",
			values: map[string]storage.Gauge{
				"gauge1": 1e10,
				"gauge2": 2e10,
			},
			expected: map[string]storage.Gauge{
				"gauge1": 1e10,
				"gauge2": 2e10,
			},
		},
		{
			name: "multiple gauges small",
			values: map[string]storage.Gauge{
				"gauge1": 1e-10,
				"gauge2": 2e-10,
			},
			expected: map[string]storage.Gauge{
				"gauge1": 1e-10,
				"gauge2": 2e-10,
			},
		},
	}

	memStorage := storage.NewMemStorage()
	poller := basePoller{
		storage: memStorage,
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var wg sync.WaitGroup
			for k, v := range tc.values {
				wg.Add(1)
				go func(key string, value storage.Gauge) {
					defer wg.Done()
					err := poller.storeGauges(context.Background(), map[string]storage.Gauge{key: value})
					if err != nil {
						t.Errorf("unexpected error: %v", err)
					}
				}(k, v)
			}
			wg.Wait()

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

func TestBasePoller_SetCounters(t *testing.T) {
	testCases := []struct {
		name     string
		values   map[string]storage.Counter
		expected map[string]storage.Counter
	}{
		{
			name: "single counter",
			values: map[string]storage.Counter{
				"counter1": 1,
			},
			expected: map[string]storage.Counter{
				"counter1": 1,
			},
		},
		{
			name: "multiple counters",
			values: map[string]storage.Counter{
				"counter1": 1,
				"counter2": 2,
			},
			expected: map[string]storage.Counter{
				"counter1": 1 + 1,
				"counter2": 2,
			},
		},
		{
			name: "multiple counters large",
			values: map[string]storage.Counter{
				"counter1": 1e10,
				"counter2": 2e10,
			},
			expected: map[string]storage.Counter{
				"counter1": 1 + 1 + 1e10,
				"counter2": 2 + 2e10,
			},
		},
		{
			name: "multiple counters small",
			values: map[string]storage.Counter{
				"counter1": 1,
				"counter2": 2,
			},
			expected: map[string]storage.Counter{
				"counter1": 1 + 1 + 1e10 + 1,
				"counter2": 2 + 2e10 + 2,
			},
		},
	}

	memStorage := storage.NewMemStorage()
	poller := basePoller{
		storage: memStorage,
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := poller.storeCounters(context.Background(), tc.values)
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

func TestBasePoller_SetCounters_Concurrent(t *testing.T) {
	testCases := []struct {
		name     string
		values   map[string]storage.Counter
		expected map[string]storage.Counter
	}{
		{
			name: "single counter",
			values: map[string]storage.Counter{
				"counter1": 1,
			},
			expected: map[string]storage.Counter{
				"counter1": 1,
			},
		},
		{
			name: "multiple counters",
			values: map[string]storage.Counter{
				"counter1": 1,
				"counter2": 2,
			},
			expected: map[string]storage.Counter{
				"counter1": 1 + 1,
				"counter2": 2,
			},
		},
		{
			name: "multiple counters large",
			values: map[string]storage.Counter{
				"counter1": 1e10,
				"counter2": 2e10,
			},
			expected: map[string]storage.Counter{
				"counter1": 1 + 1 + 1e10,
				"counter2": 2 + 2e10,
			},
		},
		{
			name: "multiple counters small",
			values: map[string]storage.Counter{
				"counter1": 1,
				"counter2": 2,
			},
			expected: map[string]storage.Counter{
				"counter1": 1 + 1 + 1e10 + 1,
				"counter2": 2 + 2e10 + 2,
			},
		},
	}

	memStorage := storage.NewMemStorage()
	poller := basePoller{
		storage: memStorage,
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var wg sync.WaitGroup
			for k, v := range tc.values {
				wg.Add(1)
				go func(key string, value storage.Counter) {
					defer wg.Done()
					err := poller.storeCounters(context.Background(), map[string]storage.Counter{key: value})
					if err != nil {
						t.Errorf("unexpected error: %v", err)
					}
				}(k, v)
			}
			wg.Wait()

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

func TestBasePoller_GetMetrics(t *testing.T) {
	testCases := []struct {
		name     string
		gauges   map[string]storage.Gauge
		counters map[string]storage.Counter
		expected []Metric
	}{
		{
			name: "single gauge and counter",
			gauges: map[string]storage.Gauge{
				"gauge1": 1.0,
			},
			counters: map[string]storage.Counter{
				"counter1": 10,
			},
			expected: []Metric{
				{ID: "gauge1", Value: float64Ptr(1.0), MType: TypeGauge},
				{ID: "counter1", Delta: int64Ptr(10), MType: TypeCounter},
			},
		},
		{
			name: "multiple gauges and counters",
			gauges: map[string]storage.Gauge{
				"gauge1": 1.0,
				"gauge2": 2.0,
			},
			counters: map[string]storage.Counter{
				"counter1": 10,
				"counter2": 20,
			},
			expected: []Metric{
				{ID: "gauge1", Value: float64Ptr(1.0), MType: TypeGauge},
				{ID: "gauge2", Value: float64Ptr(2.0), MType: TypeGauge},
				{ID: "counter1", Delta: int64Ptr(10), MType: TypeCounter},
				{ID: "counter2", Delta: int64Ptr(20), MType: TypeCounter},
			},
		},
		{
			name:     "no metrics",
			gauges:   map[string]storage.Gauge{},
			counters: map[string]storage.Counter{},
			expected: []Metric{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			memStorage := storage.NewMemStorage()
			poller := basePoller{
				storage: memStorage,
			}

			err := memStorage.SetGauges(context.Background(), tc.gauges)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			err = memStorage.SetCounters(context.Background(), tc.counters)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			metrics, err := poller.getMetrics(context.Background())
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(metrics) != len(tc.expected) {
				t.Fatalf("expected %d metrics, got %d", len(tc.expected), len(metrics))
			}
			for _, expected := range tc.expected {
				found := false
				for _, actual := range metrics {
					if actual.ID == expected.ID && actual.MType == expected.MType {
						if expected.MType == TypeGauge &&
							actual.Value != nil &&
							expected.Value != nil &&
							*actual.Value == *expected.Value {
							found = true
							break
						}
						if expected.MType == TypeCounter &&
							actual.Delta != nil &&
							expected.Delta != nil &&
							*actual.Delta == *expected.Delta {
							found = true
							break
						}
					}
				}
				if !found {
					t.Errorf("expected metric %+v not found", expected)
				}
			}
		})
	}
}

func TestBasePoller_GetMetrics_Concurrent(t *testing.T) {
	testCases := []struct {
		name     string
		gauges   map[string]storage.Gauge
		counters map[string]storage.Counter
		expected []Metric
	}{
		{
			name: "single gauge and counter",
			gauges: map[string]storage.Gauge{
				"gauge1": 1.0,
			},
			counters: map[string]storage.Counter{
				"counter1": 10,
			},
			expected: []Metric{
				{ID: "gauge1", Value: float64Ptr(1.0), MType: TypeGauge},
				{ID: "counter1", Delta: int64Ptr(10), MType: TypeCounter},
			},
		},
		{
			name: "multiple gauges and counters",
			gauges: map[string]storage.Gauge{
				"gauge1": 1.0,
				"gauge2": 2.0,
			},
			counters: map[string]storage.Counter{
				"counter1": 10,
				"counter2": 20,
			},
			expected: []Metric{
				{ID: "gauge1", Value: float64Ptr(1.0), MType: TypeGauge},
				{ID: "gauge2", Value: float64Ptr(2.0), MType: TypeGauge},
				{ID: "counter1", Delta: int64Ptr(10), MType: TypeCounter},
				{ID: "counter2", Delta: int64Ptr(20), MType: TypeCounter},
			},
		},
		{
			name:     "no metrics",
			gauges:   map[string]storage.Gauge{},
			counters: map[string]storage.Counter{},
			expected: []Metric{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			memStorage := storage.NewMemStorage()
			poller := basePoller{
				storage: memStorage,
			}

			err := memStorage.SetGauges(context.Background(), tc.gauges)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			err = memStorage.SetCounters(context.Background(), tc.counters)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			var wg sync.WaitGroup
			const numRoutines = 10
			results := make([][]Metric, numRoutines)
			errors := make([]error, numRoutines)

			for i := range make([]struct{}, numRoutines) {
				wg.Add(1)
				go func(index int) {
					defer wg.Done()
					metrics, err := poller.getMetrics(context.Background())
					results[index] = metrics
					errors[index] = err
				}(i)
			}
			wg.Wait()

			for i := range make([]struct{}, numRoutines) {
				if errors[i] != nil {
					t.Errorf("unexpected error in goroutine %d: %v", i, errors[i])
				}
				if len(results[i]) != len(tc.expected) {
					t.Errorf("goroutine %d: expected %d metrics, got %d", i, len(tc.expected), len(results[i]))
					continue
				}
				for _, expected := range tc.expected {
					found := false
					for _, actual := range results[i] {
						if actual.ID == expected.ID && actual.MType == expected.MType {
							if expected.MType == TypeGauge &&
								actual.Value != nil &&
								expected.Value != nil &&
								*actual.Value == *expected.Value {
								found = true
								break
							}
							if expected.MType == TypeCounter &&
								actual.Delta != nil &&
								expected.Delta != nil &&
								*actual.Delta == *expected.Delta {
								found = true
								break
							}
						}
					}
					if !found {
						t.Errorf("goroutine %d: expected metric %+v not found", i, expected)
					}
				}
			}
		})
	}
}

func TestBasePoller_ResetMetrics(t *testing.T) {
	memStorage := storage.NewMemStorage()
	poller := basePoller{
		storage: memStorage,
	}

	initialGauges := map[string]storage.Gauge{"gauge1": 1.0, "gauge2": 2.0}
	initialCounters := map[string]storage.Counter{"counter1": 10, "counter2": 20}
	err := memStorage.SetGauges(context.Background(), initialGauges)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	err = memStorage.SetCounters(context.Background(), initialCounters)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = poller.resetMetrics(context.Background())
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

	counters, err := memStorage.GetCounters(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(counters) != 0 {
		t.Errorf("expected no counters, got %v", counters)
	}
}

func TestBasePoller_ResetMetrics_Concurrent(t *testing.T) {
	memStorage := storage.NewMemStorage()
	poller := basePoller{
		storage: memStorage,
	}

	initialGauges := map[string]storage.Gauge{"gauge1": 1.0, "gauge2": 2.0}
	initialCounters := map[string]storage.Counter{"counter1": 10, "counter2": 20}
	err := memStorage.SetGauges(context.Background(), initialGauges)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	err = memStorage.SetCounters(context.Background(), initialCounters)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var wg sync.WaitGroup
	const numRoutines = 10
	errors := make([]error, numRoutines)

	for i := range make([]struct{}, numRoutines) {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			errors[index] = poller.resetMetrics(context.Background())
		}(i)
	}
	wg.Wait()

	for i := range make([]struct{}, numRoutines) {
		if errors[i] != nil {
			t.Errorf("unexpected error in goroutine %d: %v", i, errors[i])
		}
	}

	gauges, err := memStorage.GetGauges(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(gauges) != 0 {
		t.Errorf("expected no gauges, got %v", gauges)
	}

	counters, err := memStorage.GetCounters(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(counters) != 0 {
		t.Errorf("expected no counters, got %v", counters)
	}
}

func float64Ptr(v float64) *float64 {
	return &v
}

func int64Ptr(v int64) *int64 {
	return &v
}
