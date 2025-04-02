package middleware

import (
	"metrics/internal/utils"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestWithHashValidation(t *testing.T) {
	key := "secret"
	body := "test"
	hash := utils.GetHash(key, []byte(body))

	router := gin.Default()
	router.Use(WithHashValidation(key))
	router.POST("/", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	req.Header.Set("HashSHA256", hash)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status code to be 200, got %d", rec.Code)
	}
}

func TestWithHashHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)

	key := "secret"
	responseBody := "test"

	router := gin.Default()
	router.Use(WithHashHeader(key))
	router.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, responseBody)
	})

	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	hash := rec.Header().Get("HashSHA256")
	expectedHash := utils.GetHash(key, []byte(responseBody))

	if hash == "" {
		t.Error("hash header is empty, middleware failed to set it")
	}

	if hash != expectedHash {
		t.Errorf("expected hash to be '%s', got '%s'", expectedHash, hash)
	}
}
