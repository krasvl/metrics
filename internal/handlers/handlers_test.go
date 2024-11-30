package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
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
func TestSetMetricHandler(t *testing.T) {

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
