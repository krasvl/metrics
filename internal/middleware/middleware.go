package middleware

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"metrics/internal/utils"

	"go.uber.org/zap"
)

type gzipResponseWriter struct {
	http.ResponseWriter
	Writer io.Writer
}

func (g *gzipResponseWriter) Write(b []byte) (int, error) {
	n, err := g.Writer.Write(b)
	if err != nil {
		return n, fmt.Errorf("cant gzip body: %w", err)
	}
	return n, nil
}

func (g *gzipResponseWriter) WriteHeader(statusCode int) {
	g.ResponseWriter.WriteHeader(statusCode)
}

func (g *gzipResponseWriter) Header() http.Header {
	return g.ResponseWriter.Header()
}

func WithDecompress(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Encoding") == "gzip" {
			gz, err := gzip.NewReader(r.Body)
			if err != nil {
				http.Error(w, "cant gzip body", http.StatusInternalServerError)
				return
			}
			defer func() {
				if err := gz.Close(); err != nil {
					log.Printf("Failed to close gzip reader: %v", err)
				}
			}()
			r.Body = gz
		}
		next.ServeHTTP(w, r)
	})
}

func WithCompress(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Accept-Encoding") == "gzip" &&
			(r.Header.Get("Accept") == "application/json" || r.Header.Get("Accept") == "text/html") {
			w.Header().Set("Content-Encoding", "gzip")
			w.Header().Set("Vary", "Accept-Encoding")

			gz := gzip.NewWriter(w)
			defer func() {
				if err := gz.Close(); err != nil {
					log.Printf("Failed to close gzip writer: %v", err)
				}
			}()

			rw := &gzipResponseWriter{
				ResponseWriter: w,
				Writer:         gz,
			}

			next.ServeHTTP(rw, r)
		} else {
			next.ServeHTTP(w, r)
		}
	})
}

type logResponseWriter struct {
	http.ResponseWriter
	statusCode    int
	contentLength int
}

func (lrw *logResponseWriter) WriteHeader(code int) {
	lrw.statusCode = code
	lrw.ResponseWriter.WriteHeader(code)
}

func (lrw *logResponseWriter) Write(data []byte) (int, error) {
	size, err := lrw.ResponseWriter.Write(data)
	lrw.contentLength += size
	return size, err
}

func WithLogging(logger *zap.Logger) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rw := &logResponseWriter{ResponseWriter: w}
			h.ServeHTTP(rw, r)
			duration := time.Since(start)

			logger.Info("Request processed",
				zap.String("uri", r.RequestURI),
				zap.String("method", r.Method),
				zap.Duration("duration", duration),
				zap.Int("status", rw.statusCode),
				zap.Int("content_length", rw.contentLength),
			)
		})
	}
}

type hashResponseWriter struct {
	http.ResponseWriter
	writer io.Writer
}

func (rw *hashResponseWriter) Write(p []byte) (int, error) {
	return rw.writer.Write(p)
}

func WithHashValidation(key string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if key == "" || r.Header.Get("HashSHA256") == "" {
				next.ServeHTTP(w, r)
				return
			}

			body, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, "cant read request body", http.StatusInternalServerError)
				return
			}
			r.Body = io.NopCloser(bytes.NewBuffer(body))

			expectedHash := utils.GetHash(key, body)
			receivedHash := r.Header.Get("HashSHA256")

			if receivedHash != expectedHash {
				http.Error(w, "invalid hash", http.StatusBadRequest)
				zap.L().Error("Hash mismatch",
					zap.String("expected", expectedHash),
					zap.String("received", receivedHash),
					zap.ByteString("body", body),
				)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func WithHashHeader(key string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			buf := new(bytes.Buffer)
			mw := io.MultiWriter(w, buf)
			rw := &hashResponseWriter{ResponseWriter: w, writer: mw}

			next.ServeHTTP(rw, r)

			if key != "" && len(buf.Bytes()) > 0 {
				hash := utils.GetHash(key, buf.Bytes())
				w.Header().Set("HashSHA256", hash)
			}
		})
	}
}
