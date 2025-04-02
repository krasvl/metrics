package middleware

import (
	"compress/gzip"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

// WithDecompress is a middleware that decompresses gzip-encoded request bodies.
func WithDecompress() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.GetHeader("Content-Encoding") == "gzip" {
			gz, err := gzip.NewReader(c.Request.Body)
			if err != nil {
				c.String(http.StatusInternalServerError, "cant gzip body")
				c.Abort()
				return
			}
			defer func() {
				if err := gz.Close(); err != nil {
					log.Printf("Failed to close gzip reader: %v", err)
				}
			}()
			c.Request.Body = io.NopCloser(gz)
		}
		c.Next()
	}
}

// WithCompress is a middleware that compresses response bodies using gzip if the client supports it.
func WithCompress() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.GetHeader("Accept-Encoding") == "gzip" &&
			(c.GetHeader("Accept") == "application/json" || c.GetHeader("Accept") == "text/html") {
			c.Header("Content-Encoding", "gzip")
			c.Header("Vary", "Accept-Encoding")

			originalWriter := c.Writer
			gz := gzip.NewWriter(originalWriter)

			c.Writer = &gzipResponseWriter{
				ResponseWriter: originalWriter,
				Writer:         gz,
			}

			defer func() {
				if err := gz.Close(); err != nil {
					log.Printf("Failed to close gzip writer: %v", err)
				}
			}()
		}
		c.Next()
	}
}

type gzipResponseWriter struct {
	gin.ResponseWriter
	Writer io.Writer
}

func (g *gzipResponseWriter) WriteString(s string) (int, error) {
	if sw, ok := g.Writer.(io.StringWriter); ok {
		return sw.WriteString(s)
	}
	return g.Writer.Write([]byte(s))
}

func (g *gzipResponseWriter) Write(b []byte) (int, error) {
	n, err := g.Writer.Write(b)
	if err != nil {
		return n, fmt.Errorf("cant gzip body: %w", err)
	}
	return n, nil
}
