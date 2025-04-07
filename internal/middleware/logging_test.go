package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func TestWithLogging(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	router := gin.Default()
	router.Use(WithLogging(logger))
	router.GET("/", func(c *gin.Context) {
		c.Status(http.StatusOK)
		if _, err := c.Writer.WriteString("test"); err != nil {
			t.Errorf("Failed to write: %v", err)
		}
	})

	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status code to be 200, got %d", rec.Code)
	}
}
