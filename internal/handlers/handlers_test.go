package handlers

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"metrics/internal/storage"
)

func TestSetGaugeMetricHandler(t *testing.T) {

	gaugeTests := []struct {
		url            string
		metricName     string
		metricValue    string
		expectedStatus int
	}{
		{"/update/gauge/{metricName}/{metricName}", "test1", "0", http.StatusOK},
		{"/update/gauge/{metricName}/{metricName}", "test1", "-1", http.StatusOK},
		{"/update/gauge/{metricName}/{metricName}", "test1", "1.1", http.StatusOK},
		{"/update/gauge/{metricName}/{metricName}", "test1", "no", http.StatusBadRequest},
		{"/update/gauge/{metricName}/{metricName}", "test1", "", http.StatusBadRequest},
	}

	memStorage := storage.NewMemStorage()
	handler := NewMetricsHandler(memStorage)

	for _, tt := range gaugeTests {
		r := httptest.NewRequest("POST", tt.url, nil)
		w := httptest.NewRecorder()

		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("metricName", tt.metricName)
		rctx.URLParams.Add("metricValue", tt.metricValue)
		r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))

		handler.SetGaugeMetricHandler(w, r)

		res := w.Result()
		defer res.Body.Close()

		if res.StatusCode != tt.expectedStatus {
			t.Errorf("expected status %d, got %d, url %s", tt.expectedStatus, res.StatusCode, tt.url)
		}
	}
}

func TestSetCounterMetricHandler(t *testing.T) {

	counterTests := []struct {
		url            string
		metricName     string
		metricValue    string
		expectedStatus int
	}{
		{"/update/counter/{metricName}/{metricName}", "test1", "0", http.StatusOK},
		{"/update/counter/{metricName}/{metricName}", "test1", "-1", http.StatusOK},
		{"/update/counter/{metricName}/{metricName}", "test1", "1.1", http.StatusBadRequest},
		{"/update/couner/{metricName}/{metricName}", "test1", "no", http.StatusBadRequest},
		{"/update/counter/{metricName}/{metricName}", "test1", "", http.StatusBadRequest},
	}

	memStorage := storage.NewMemStorage()
	handler := NewMetricsHandler(memStorage)

	for _, tt := range counterTests {
		r := httptest.NewRequest("POST", tt.url, nil)
		w := httptest.NewRecorder()

		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("metricName", tt.metricName)
		rctx.URLParams.Add("metricValue", tt.metricValue)
		r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))

		handler.SetCounterMetricHandler(w, r)

		res := w.Result()
		defer res.Body.Close()

		if res.StatusCode != tt.expectedStatus {
			t.Errorf("expected status %d, got %d, url %s", tt.expectedStatus, res.StatusCode, tt.url)
		}
	}
}

func TestGetGaugeMetricHandler(t *testing.T) {
	gaugeTests := []struct {
		metricName     string
		metricValue    float64
		expectedStatus int
		expectedBody   string
	}{
		{"test1", 1.1, http.StatusOK, "1.1"},
		{"test2", 0, http.StatusOK, "0"},
		{"test3", -1, http.StatusOK, "-1"},
		{"nonexistent", 0, http.StatusNotFound, ""},
	}

	memStorage := storage.NewMemStorage()
	handler := NewMetricsHandler(memStorage)

	for _, tt := range gaugeTests {
		if tt.metricName != "nonexistent" {
			memStorage.SetGauge(tt.metricName, storage.Gauge(tt.metricValue))
		}

		r := httptest.NewRequest("GET", "/value/gauge/{metricName}", nil)
		w := httptest.NewRecorder()

		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("metricName", tt.metricName)
		r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))

		handler.GetGaugeMetricHandler(w, r)

		res := w.Result()
		defer res.Body.Close()

		if res.StatusCode != tt.expectedStatus {
			t.Errorf("expected status %d, got %d for metric %s", tt.expectedStatus, res.StatusCode, tt.metricName)
		}

		if tt.expectedStatus == http.StatusOK {
			body, _ := io.ReadAll(res.Body)
			if string(body) != tt.expectedBody {
				t.Errorf("expected body %s, got %s for metric %s", tt.expectedBody, string(body), tt.metricName)
			}
		}
	}
}

func TestGetCounterMetricHandler(t *testing.T) {
	counterTests := []struct {
		metricName     string
		metricValue    int
		expectedStatus int
		expectedBody   string
	}{
		{"test1", 1, http.StatusOK, "1"},
		{"test2", 0, http.StatusOK, "0"},
		{"test3", -1, http.StatusOK, "-1"},
		{"nonexistent", 0, http.StatusNotFound, ""},
	}

	memStorage := storage.NewMemStorage()
	handler := NewMetricsHandler(memStorage)

	for _, tt := range counterTests {
		if tt.metricName != "nonexistent" {
			memStorage.SetCounter(tt.metricName, storage.Counter(tt.metricValue))
		}

		r := httptest.NewRequest("GET", "/value/gauge/{metricName}", nil)
		w := httptest.NewRecorder()

		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("metricName", tt.metricName)
		r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))

		handler.GetCounterMetricHandler(w, r)

		res := w.Result()
		defer res.Body.Close()

		if res.StatusCode != tt.expectedStatus {
			t.Errorf("expected status %d, got %d for metric %s", tt.expectedStatus, res.StatusCode, tt.metricName)
		}

		if tt.expectedStatus == http.StatusOK {
			body, _ := io.ReadAll(res.Body)
			if string(body) != tt.expectedBody {
				t.Errorf("expected body %s, got %s for metric %s", tt.expectedBody, string(body), tt.metricName)
			}
		}
	}
}

func TestGetAllMetricsHandler(t *testing.T) {
	gaugeTests := []struct {
		metricName  string
		metricValue float64
	}{
		{"gauge1", 1.1},
		{"gauge2", -1.0},
	}
	counterTests := []struct {
		metricName  string
		metricValue int64
	}{
		{"counter1", 1},
		{"counter2", 0},
	}

	memStorage := storage.NewMemStorage()
	handler := NewMetricsHandler(memStorage)

	for _, tt := range gaugeTests {
		memStorage.SetGauge(tt.metricName, storage.Gauge(tt.metricValue))
	}

	for _, tt := range counterTests {
		memStorage.SetCounter(tt.metricName, storage.Counter(tt.metricValue))
	}

	r := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	handler.GetAllMetricsHandler(w, r)

	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, res.StatusCode)
	}

	body, _ := io.ReadAll(res.Body)
	bodyStr := string(body)

	for _, tt := range gaugeTests {
		if !strings.Contains(bodyStr, tt.metricName) {
			t.Errorf("expected body to contain metric %s, but it didn't", tt.metricName)
		}
	}
	for _, tt := range counterTests {
		if !strings.Contains(bodyStr, tt.metricName) {
			t.Errorf("expected body to contain metric %s, but it didn't", tt.metricName)
		}
	}
}
