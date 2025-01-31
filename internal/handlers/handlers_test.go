package handlers

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"metrics/internal/storage"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

type testCase struct {
	method         string
	url            string
	routeParams    map[string]string
	body           interface{}
	expectedStatus int
	expectedBody   string
}

func createRequest(method, url string, routeParams map[string]string) *http.Request {
	r := httptest.NewRequest(method, url, http.NoBody)
	rctx := chi.NewRouteContext()
	for key, value := range routeParams {
		rctx.URLParams.Add(key, value)
	}
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

func runTests(t *testing.T, handlerFunc http.HandlerFunc, tests []testCase) {
	t.Helper()

	for _, tt := range tests {
		r := createRequest(tt.method, tt.url, tt.routeParams)
		w := httptest.NewRecorder()

		handlerFunc(w, r)

		res := w.Result()

		if res.StatusCode != tt.expectedStatus {
			t.Errorf("Expected status %d, got %d", tt.expectedStatus, res.StatusCode)
			return
		}

		if tt.expectedStatus == http.StatusOK {
			body, err := io.ReadAll(res.Body)
			if err != nil {
				t.Fatalf("Cant read response body: %v", err)
			}
			if string(body) != tt.expectedBody {
				t.Errorf("Expected body %s, got %s", tt.expectedBody, string(body))
			}
		}

		if err := res.Body.Close(); err != nil {
			t.Fatalf("Cant close response body: %v", err)
		}
	}
}

func TestSetGaugeMetricHandler(t *testing.T) {
	gaugeTests := []testCase{
		{
			"POST",
			"/update/gauge/{metricName}/{metricValue}",
			map[string]string{"metricName": "gaugeTest", "metricValue": "0"},
			nil,
			http.StatusOK,
			"",
		},
		{
			"POST",
			"/update/gauge/{metricName}/{metricValue}",
			map[string]string{"metricName": "gaugeTest", "metricValue": "-100000"},
			nil,
			http.StatusOK,
			""},
		{
			"POST", "/update/gauge/{metricName}/{metricValue}",
			map[string]string{"metricName": "gaugeTest", "metricValue": "1.1432423"},
			nil,
			http.StatusOK,
			"",
		},
		{
			"POST",
			"/update/gauge/{metricName}/{metricValue}",
			map[string]string{"metricName": "gaugeTest", "metricValue": "gaugeTestBad"},
			nil,
			http.StatusBadRequest,
			"",
		},
		{
			"POST",
			"/update/gauge/{metricName}/{metricValue}",
			map[string]string{"metricName": "gaugeTest", "metricValue": ""},
			nil,
			http.StatusBadRequest,
			"",
		},
	}

	memStorage := storage.NewMemStorage()
	logger, _ := zap.NewProduction()
	handler := NewMetricsHandler(memStorage, logger)

	runTests(t, handler.SetGaugeMetricHandler, gaugeTests)
}

func TestSetCounterMetricHandler(t *testing.T) {
	counterTests := []testCase{
		{
			"POST",
			"/update/counter/{metricName}/{metricValue}",
			map[string]string{"metricName": "counterTest", "metricValue": "0"},
			nil,
			http.StatusOK,
			"",
		},
		{
			"POST",
			"/update/counter/{metricName}/{metricValue}",
			map[string]string{"metricName": "counterTest", "metricValue": "-123456"},
			nil,
			http.StatusOK,
			"",
		},
		{
			"POST",
			"/update/counter/{metricName}/{metricValue}",
			map[string]string{"metricName": "counterTest", "metricValue": "3.1123"},
			nil,
			http.StatusBadRequest,
			"",
		},
		{
			"POST",
			"/update/counter/{metricName}/{metricValue}",
			map[string]string{"metricName": "counterTest", "metricValue": "counterTestBad"},
			nil,
			http.StatusBadRequest,
			"",
		},
		{
			"POST",
			"/update/counter/{metricName}/{metricValue}",
			map[string]string{"metricName": "counterTest", "metricValue": ""},
			nil,
			http.StatusBadRequest,
			"",
		},
	}

	memStorage := storage.NewMemStorage()
	logger, _ := zap.NewProduction()
	handler := NewMetricsHandler(memStorage, logger)

	runTests(t, handler.SetCounterMetricHandler, counterTests)
}

func TestGetGaugeMetricHandler(t *testing.T) {
	tests := []testCase{
		{
			"GET", "/value/gauge/{metricName}",
			map[string]string{"metricName": "test"},
			nil,
			http.StatusOK,
			"1",
		},
		{
			"GET", "/value/gauge/{metricName}",
			map[string]string{"metricName": "noexist"},
			nil,
			http.StatusNotFound,
			"",
		},
	}

	memStorage := storage.NewMemStorage()
	logger, _ := zap.NewProduction()
	ctx := context.Background()
	_ = memStorage.SetGauge(ctx, "test", storage.Gauge(1))

	handler := NewMetricsHandler(memStorage, logger)

	runTests(t, handler.GetGaugeMetricHandler, tests)
}

func TestGetCounterMetricHandler(t *testing.T) {
	tests := []testCase{
		{
			"GET",
			"/value/counter/{metricName}",
			map[string]string{"metricName": "test"},
			nil,
			http.StatusOK,
			"1",
		},
		{
			"GET", "/value/counter/{metricName}",
			map[string]string{"metricName": "noexist"},
			nil,
			http.StatusNotFound,
			"",
		},
	}

	memStorage := storage.NewMemStorage()
	logger, _ := zap.NewProduction()
	ctx := context.Background()
	_ = memStorage.SetCounter(ctx, "test", storage.Counter(1))

	handler := NewMetricsHandler(memStorage, logger)

	runTests(t, handler.GetCounterMetricHandler, tests)
}

func TestGetAllMetricsHandler(t *testing.T) {
	body := `
<!DOCTYPE html>
<html>
<head>
    <title>Metrics</title>
</head>
<body>
    <h1>Metrics</h1>
    <h2>Gauges</h2>
    <ul>
        <li>testGauge1: 1.1</li>
        <li>testGauge2: 1.2</li>
    </ul>
    <h2>Counters</h2>
    <ul>
        <li>testCounter: 2</li>
    </ul>
</body>
</html>
	`

	tests := []testCase{
		{"GET", "/", map[string]string{}, nil, http.StatusOK, body},
	}

	memStorage := storage.NewMemStorage()
	logger, _ := zap.NewProduction()
	ctx := context.Background()
	_ = memStorage.SetGauge(ctx, "testGauge1", storage.Gauge(1.1))
	_ = memStorage.SetGauge(ctx, "testGauge2", storage.Gauge(1.2))
	_ = memStorage.SetCounter(ctx, "testCounter", storage.Counter(1))
	_ = memStorage.SetCounter(ctx, "testCounter", storage.Counter(1))

	handler := NewMetricsHandler(memStorage, logger)

	runTests(t, handler.GetMetricsReportHandler, tests)
}
