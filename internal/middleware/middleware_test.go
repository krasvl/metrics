package middleware

import (
	"bytes"
	"compress/gzip"
	"io"
	"metrics/internal/utils"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.uber.org/zap"
)

func TestWithDecompress(t *testing.T) {
	handler := WithDecompress(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if string(body) != "test" {
			t.Errorf("expected body to be 'test', got '%s'", string(body))
		}
	}))

	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	_, _ = gz.Write([]byte("test"))
	if err := gz.Close(); err != nil {
		t.Errorf("Failed to close gzip: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/", &buf)
	req.Header.Set("Content-Encoding", "gzip")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)
}

func TestWithCompress(t *testing.T) {
	handler := WithCompress(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := w.Write([]byte("test")); err != nil {
			t.Errorf("Failed to write: %v", err)
		}
	}))

	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	req.Header.Set("Accept-Encoding", "gzip")
	req.Header.Set("Accept", "application/json")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	resp := rec.Result()
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Errorf("Failed to close response body: %v", err)
		}
	}()

	if resp.Header.Get("Content-Encoding") != "gzip" {
		t.Errorf("expected Content-Encoding to be 'gzip', got '%s'", resp.Header.Get("Content-Encoding"))
	}

	if resp.Header.Get("Vary") != "Accept-Encoding" {
		t.Errorf("expected Vary header to be 'Accept-Encoding', got '%s'", resp.Header.Get("Vary"))
	}

	gz, err := gzip.NewReader(resp.Body)
	if err != nil {
		t.Fatalf("failed to create gzip reader: %v", err)
	}
	defer func() {
		if err := gz.Close(); err != nil {
			t.Errorf("Failed to close gzip: %v", err)
		}
	}()

	body, err := io.ReadAll(gz)
	if err != nil {
		t.Fatalf("failed to read gzip body: %v", err)
	}

	if string(body) != "test" {
		t.Errorf("expected body to be 'test', got '%s'", string(body))
	}
}

func TestWithLogging(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	handler := WithLogging(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("test")); err != nil {
			t.Errorf("Failed to write: %v", err)
		}
	}))

	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status code to be 200, got %d", rec.Code)
	}
}

func TestWithHashValidation(t *testing.T) {
	key := "secret"
	body := "test"
	hash := utils.GetHash(key, []byte(body))

	handler := WithHashValidation(key)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	req.Header.Set("HashSHA256", hash)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status code to be 200, got %d", rec.Code)
	}
}

func TestWithHashHeader(t *testing.T) {
	key := "secret"
	handler := WithHashHeader(key)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := w.Write([]byte("test")); err != nil {
			t.Errorf("Failed to write: %v", err)
		}
	}))

	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	hash := rec.Header().Get("HashSHA256")
	expectedHash := utils.GetHash(key, []byte("test"))
	if hash != expectedHash {
		t.Errorf("expected hash to be '%s', got '%s'", expectedHash, hash)
	}
}
