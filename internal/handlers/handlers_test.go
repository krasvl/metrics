package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"metrics/internal/storage"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestSetGaugeMetricHandler(t *testing.T) {
	testCases := []struct {
		name          string
		metricName    string
		metricValue   string
		expectedCode  int
		expectedValue storage.Gauge
		expectError   bool
	}{
		{
			name:          "valid gauge metric",
			metricName:    "testMetric",
			metricValue:   "123.45",
			expectedCode:  http.StatusOK,
			expectedValue: 123.45,
			expectError:   false,
		},
		{
			name:         "invalid gauge value",
			metricName:   "testMetric",
			metricValue:  "invalid",
			expectedCode: http.StatusBadRequest,
			expectError:  true,
		},
		{
			name:         "missing metric name",
			metricName:   "",
			metricValue:  "123.45",
			expectedCode: http.StatusNotFound,
			expectError:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockStorage := storage.NewMemStorage()
			logger := zap.NewNop()
			handler := NewMetricsHandler(mockStorage, logger)

			req := httptest.NewRequest(http.MethodPost, "/update/gauge/"+tc.metricName+"/"+tc.metricValue, http.NoBody)
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("metricName", tc.metricName)
			rctx.URLParams.Add("metricValue", tc.metricValue)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			w := httptest.NewRecorder()
			handler.SetGaugeMetricHandler(w, req)

			assert.Equal(t, tc.expectedCode, w.Code)
			if !tc.expectError {
				value, ok, _ := mockStorage.GetGauge(req.Context(), tc.metricName)
				assert.True(t, ok)
				assert.Equal(t, tc.expectedValue, value)
			}
		})
	}
}

func TestSetCounterMetricHandler(t *testing.T) {
	testCases := []struct {
		name          string
		metricName    string
		metricValue   string
		expectedCode  int
		expectedValue storage.Counter
		expectError   bool
	}{
		{
			name:          "valid counter metric",
			metricName:    "testMetric",
			metricValue:   "10",
			expectedCode:  http.StatusOK,
			expectedValue: 10,
			expectError:   false,
		},
		{
			name:         "invalid counter value",
			metricName:   "testMetric",
			metricValue:  "invalid",
			expectedCode: http.StatusBadRequest,
			expectError:  true,
		},
		{
			name:         "missing metric name",
			metricName:   "",
			metricValue:  "10",
			expectedCode: http.StatusNotFound,
			expectError:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockStorage := storage.NewMemStorage()
			logger := zap.NewNop()
			handler := NewMetricsHandler(mockStorage, logger)

			req := httptest.NewRequest(http.MethodPost, "/update/counter/"+tc.metricName+"/"+tc.metricValue, http.NoBody)
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("metricName", tc.metricName)
			rctx.URLParams.Add("metricValue", tc.metricValue)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			w := httptest.NewRecorder()
			handler.SetCounterMetricHandler(w, req)

			assert.Equal(t, tc.expectedCode, w.Code)
			if !tc.expectError {
				value, ok, _ := mockStorage.GetCounter(req.Context(), tc.metricName)
				assert.True(t, ok)
				assert.Equal(t, tc.expectedValue, value)
			}
		})
	}
}

func TestGetGaugeMetricHandler(t *testing.T) {
	testCases := []struct {
		name         string
		metricName   string
		setupValue   storage.Gauge
		expectedCode int
		expectedBody string
		expectError  bool
	}{
		{
			name:         "existing gauge metric",
			metricName:   "testMetric",
			setupValue:   123.45,
			expectedCode: http.StatusOK,
			expectedBody: "123.45",
			expectError:  false,
		},
		{
			name:         "non-existing gauge metric",
			metricName:   "nonExistent",
			expectedCode: http.StatusNotFound,
			expectError:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockStorage := storage.NewMemStorage()
			logger := zap.NewNop()
			handler := NewMetricsHandler(mockStorage, logger)

			if tc.setupValue != 0 {
				if err := mockStorage.SetGauge(context.TODO(), tc.metricName, tc.setupValue); err != nil {
					t.Errorf("Failed to set gauge: %v", err)
				}
			}

			req := httptest.NewRequest(http.MethodGet, "/value/gauge/"+tc.metricName, http.NoBody)
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("metricName", tc.metricName)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			w := httptest.NewRecorder()
			handler.GetGaugeMetricHandler(w, req)

			assert.Equal(t, tc.expectedCode, w.Code)
			if !tc.expectError {
				assert.Equal(t, tc.expectedBody, w.Body.String())
			}
		})
	}
}

func TestGetCounterMetricHandler(t *testing.T) {
	testCases := []struct {
		name         string
		metricName   string
		setupValue   storage.Counter
		expectedCode int
		expectedBody string
		expectError  bool
	}{
		{
			name:         "existing counter metric",
			metricName:   "testMetric",
			setupValue:   10,
			expectedCode: http.StatusOK,
			expectedBody: "10",
			expectError:  false,
		},
		{
			name:         "non-existing counter metric",
			metricName:   "nonExistent",
			expectedCode: http.StatusNotFound,
			expectError:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockStorage := storage.NewMemStorage()
			logger := zap.NewNop()
			handler := NewMetricsHandler(mockStorage, logger)

			if tc.setupValue != 0 {
				if err := mockStorage.SetCounter(context.TODO(), tc.metricName, tc.setupValue); err != nil {
					t.Errorf("Failed to set counter: %v", err)
				}
			}

			req := httptest.NewRequest(http.MethodGet, "/value/counter/"+tc.metricName, http.NoBody)
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("metricName", tc.metricName)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			w := httptest.NewRecorder()
			handler.GetCounterMetricHandler(w, req)

			assert.Equal(t, tc.expectedCode, w.Code)
			if !tc.expectError {
				assert.Equal(t, tc.expectedBody, w.Body.String())
			}
		})
	}
}

func TestSetMetricHandlers(t *testing.T) {
	mockStorage := storage.NewMemStorage()
	logger := zap.NewNop()
	handler := NewMetricsHandler(mockStorage, logger)

	testCases := []struct {
		name         string
		url          string
		body         []byte
		contentType  string
		expectedCode int
		expectedBody string
		setup        func()
		verify       func(t *testing.T)
	}{
		{
			name: "valid gauge metric",
			url:  "/update/",
			body: func() []byte {
				b, _ := json.Marshal(Metric{ID: "testMetric", MType: "gauge", Value: func() *float64 { v := 123.45; return &v }()})
				return b
			}(),
			contentType:  "application/json",
			expectedCode: http.StatusOK,
			verify: func(t *testing.T) {
				t.Helper()
				value, ok, _ := mockStorage.GetGauge(context.Background(), "testMetric")
				assert.True(t, ok)
				assert.Equal(t, storage.Gauge(123.45), value)
			},
		},
		{
			name: "valid metrics batch",
			url:  "/updates/",
			body: func() []byte {
				b, _ := json.Marshal([]Metric{
					{ID: "gauge1", MType: "gauge", Value: func() *float64 { v := 123.45; return &v }()},
					{ID: "counter1", MType: "counter", Delta: func() *int64 { d := int64(10); return &d }()},
				})
				return b
			}(),
			contentType:  "application/json",
			expectedCode: http.StatusOK,
			verify: func(t *testing.T) {
				t.Helper()
				value, ok, _ := mockStorage.GetGauge(context.Background(), "gauge1")
				assert.True(t, ok)
				assert.Equal(t, storage.Gauge(123.45), value)

				counter, ok, _ := mockStorage.GetCounter(context.Background(), "counter1")
				assert.True(t, ok)
				assert.Equal(t, storage.Counter(10), counter)
			},
		},
		{
			name:         "invalid JSON",
			url:          "/update/",
			body:         []byte("invalid json"),
			contentType:  "application/json",
			expectedCode: http.StatusBadRequest,
			expectedBody: "Bad json",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.setup != nil {
				tc.setup()
			}

			req := httptest.NewRequest(http.MethodPost, tc.url, bytes.NewReader(tc.body))
			req.Header.Set("Content-Type", tc.contentType)

			w := httptest.NewRecorder()
			if tc.url == "/update/" {
				handler.SetMetricHandler(w, req)
			} else {
				handler.SetMetricsHandler(w, req)
			}

			assert.Equal(t, tc.expectedCode, w.Code)
			if tc.expectedBody != "" {
				assert.Contains(t, w.Body.String(), tc.expectedBody)
			}
			if tc.verify != nil {
				tc.verify(t)
			}
		})
	}
}

func TestGetMetricsHandler(t *testing.T) {
	mockStorage := storage.NewMemStorage()
	logger := zap.NewNop()
	handler := NewMetricsHandler(mockStorage, logger)

	if err := mockStorage.SetGauge(context.TODO(), "gauge1", 123.45); err != nil {
		t.Fatalf("Failed to set gauge: %v", err)
	}
	if err := mockStorage.SetCounter(context.TODO(), "counter1", 10); err != nil {
		t.Fatalf("Failed to set counter: %v", err)
	}

	testCases := []struct {
		name         string
		metric       Metric
		expectedCode int
		expectedBody Metric
		expectError  bool
	}{
		{
			name:         "existing gauge metric",
			metric:       Metric{ID: "gauge1", MType: "gauge"},
			expectedCode: http.StatusOK,
			expectedBody: Metric{ID: "gauge1", MType: "gauge", Value: new(float64)},
			expectError:  false,
		},
		{
			name:         "existing counter metric",
			metric:       Metric{ID: "counter1", MType: "counter"},
			expectedCode: http.StatusOK,
			expectedBody: Metric{ID: "counter1", MType: "counter", Delta: new(int64)},
			expectError:  false,
		},
		{
			name:         "non-existing metric",
			metric:       Metric{ID: "nonExistent", MType: "gauge"},
			expectedCode: http.StatusNotFound,
			expectError:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.expectedBody.Value != nil {
				*tc.expectedBody.Value = 123.45
			}
			if tc.expectedBody.Delta != nil {
				*tc.expectedBody.Delta = 10
			}

			body, _ := json.Marshal(tc.metric)
			req := httptest.NewRequest(http.MethodPost, "/value/", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			handler.GetMetricsHandler(w, req)

			assert.Equal(t, tc.expectedCode, w.Code)
			if !tc.expectError {
				var response Metric
				err := json.NewDecoder(w.Body).Decode(&response)
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedBody, response)
			}
		})
	}
}

func TestGetMetricsHandler_InvalidJSON(t *testing.T) {
	mockStorage := storage.NewMemStorage()
	logger := zap.NewNop()
	handler := NewMetricsHandler(mockStorage, logger)

	req := httptest.NewRequest(http.MethodPost, "/value/", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	handler.GetMetricsHandler(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "Bad json")
}

func TestGetMetricsHandler_UnsupportedContentType(t *testing.T) {
	mockStorage := storage.NewMemStorage()
	logger := zap.NewNop()
	handler := NewMetricsHandler(mockStorage, logger)

	req := httptest.NewRequest(http.MethodPost, "/value/", bytes.NewReader([]byte("{}")))
	req.Header.Set("Content-Type", "text/plain")

	w := httptest.NewRecorder()
	handler.GetMetricsHandler(w, req)

	assert.Equal(t, http.StatusUnsupportedMediaType, w.Code)
	assert.Contains(t, w.Body.String(), "Unsupported Content-Type")
}

func TestGetMetricsReportHandler(t *testing.T) {
	mockStorage := storage.NewMemStorage()
	logger := zap.NewNop()
	handler := NewMetricsHandler(mockStorage, logger)

	if err := mockStorage.SetGauge(context.TODO(), "gauge1", 123.45); err != nil {
		t.Fatalf("Failed to set gauge: %v", err)
	}
	if err := mockStorage.SetCounter(context.TODO(), "counter1", 10); err != nil {
		t.Fatalf("Failed to set counter: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/report/", http.NoBody)
	w := httptest.NewRecorder()
	handler.GetMetricsReportHandler(w, req)

	expectedHTML := `
<!DOCTYPE html>
<html>
<head>
    <title>Metrics</title>
</head>
<body>
    <h1>Metrics</h1>
    <h2>Gauges</h2>
    <ul>
        <li>gauge1: 123.45</li>
    </ul>
    <h2>Counters</h2>
    <ul>
        <li>counter1: 10</li>
    </ul>
</body>
</html>
`

	actualHTML := w.Body.String()
	actualHTML = actualHTML[:len(actualHTML)-1]
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, expectedHTML, actualHTML, "HTML output mismatch")
}

func TestPingHandler(t *testing.T) {
	mockStorage := storage.NewMemStorage()
	logger := zap.NewNop()
	handler := NewMetricsHandler(mockStorage, logger)

	req := httptest.NewRequest(http.MethodGet, "/ping", http.NoBody)
	w := httptest.NewRecorder()
	handler.PingHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}
