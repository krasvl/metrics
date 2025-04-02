package middleware

import (
	"bytes"
	"io"
	"net/http"

	"metrics/internal/utils"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// WithHashValidation is a middleware that validates the hash of the request body using a shared key.
func WithHashValidation(key string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if key == "" || c.GetHeader("HashSHA256") == "" {
			c.Next()
			return
		}

		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.String(http.StatusInternalServerError, "cant read request body")
			c.Abort()
			return
		}
		c.Request.Body = io.NopCloser(bytes.NewBuffer(body))

		expectedHash := utils.GetHash(key, body)
		receivedHash := c.GetHeader("HashSHA256")

		if receivedHash != expectedHash {
			c.String(http.StatusBadRequest, "invalid hash")
			zap.L().Error("Hash mismatch",
				zap.String("expected", expectedHash),
				zap.String("received", receivedHash),
				zap.ByteString("body", body),
			)
			c.Abort()
			return
		}

		c.Next()
	}
}

// WithHashHeader is a middleware that adds a hash of the response body to the response headers.
func WithHashHeader(key string) gin.HandlerFunc {
	return func(c *gin.Context) {
		buf := new(bytes.Buffer)
		mw := io.MultiWriter(c.Writer, buf)
		rw := &hashResponseWriter{ResponseWriter: c.Writer, writer: mw}

		c.Writer = rw
		c.Next()

		if key != "" && len(buf.Bytes()) > 0 {
			hash := utils.GetHash(key, buf.Bytes())
			c.Header("HashSHA256", hash)
		}
	}
}

type hashResponseWriter struct {
	gin.ResponseWriter
	writer io.Writer
}

func (rw *hashResponseWriter) Write(p []byte) (int, error) {
	return rw.writer.Write(p)
}
