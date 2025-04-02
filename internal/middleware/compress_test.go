package middleware

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestWithDecompress(t *testing.T) {
	router := gin.Default()
	router.Use(WithDecompress())
	router.POST("/", func(c *gin.Context) {
		body, _ := io.ReadAll(c.Request.Body)
		if string(body) != "test" {
			t.Errorf("expected body to be 'test', got '%s'", string(body))
		}
	})

	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	_, _ = gz.Write([]byte("test"))
	if err := gz.Close(); err != nil {
		t.Errorf("Failed to close gzip: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/", &buf)
	req.Header.Set("Content-Encoding", "gzip")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
}

func TestWithCompress(t *testing.T) {
	router := gin.Default()
	router.Use(WithCompress())
	router.GET("/", func(c *gin.Context) {
		if _, err := c.Writer.WriteString("test"); err != nil {
			t.Errorf("Failed to write: %v", err)
		}
	})

	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	req.Header.Set("Accept-Encoding", "gzip")
	req.Header.Set("Accept", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

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
