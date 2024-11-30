package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"metrics/internal/storage"
)

func TestSetMetricHandler(t *testing.T) {

	tests := []struct {
		method         string
		url            string
		expectedStatus int
	}{
		{"POST", "/update/gauge/test1/1", http.StatusOK},
		{"POST", "/update/gauge/test1/0", http.StatusOK},
		{"POST", "/update/gauge/test1/-1", http.StatusOK},
		{"POST", "/update/gauge/test1/1.1", http.StatusOK},
		{"POST", "/update/gauge/test1/", http.StatusBadRequest},
		{"POST", "/update/gauge/test1/no", http.StatusBadRequest},
		{"POST", "/update/gauge/", http.StatusNotFound},

		{"POST", "/update/counter/test1/1", http.StatusOK},
		{"POST", "/update/counter/test1/0", http.StatusOK},
		{"POST", "/update/counter/test1/-1", http.StatusOK},
		{"POST", "/update/counter/test1/1.1", http.StatusBadRequest},
		{"POST", "/update/couner/test1/", http.StatusBadRequest},
		{"POST", "/update/counter/test1/no", http.StatusBadRequest},
		{"POST", "/update/counter/", http.StatusNotFound},

		{"POST", "/update/no/", http.StatusBadRequest},
	}

	memStorage := storage.NewMemStorage()
	srv := NewServer(memStorage)

	for _, tt := range tests {
		req := httptest.NewRequest(tt.method, tt.url, nil)
		w := httptest.NewRecorder()

		srv.handler.SetMetricHandler(w, req)

		res := w.Result()
		if res.StatusCode != tt.expectedStatus {
			t.Errorf("expected status %d, got %d, url %s", tt.expectedStatus, res.StatusCode, tt.url)
		}
	}
}
