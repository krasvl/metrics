package agent

import (
	"context"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"metrics/internal/pollers"
	"metrics/internal/storage"

	"go.uber.org/zap"
)

func BenchmarkAgent_Run_1Push(b *testing.B) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/ping" {
			w.WriteHeader(http.StatusOK)
			return
		}
		if r.URL.Path == "/updates/" {
			w.WriteHeader(http.StatusOK)
			if _, err := w.Write([]byte(`{"status": "success"}`)); err != nil {
				log.Println("write error:", err)
			}
			return
		}
		http.NotFound(w, r)
	}))
	defer mockServer.Close()

	logger := zap.NewNop()
	config := Config{
		ServerURL:    mockServer.URL,
		Key:          "test-key",
		PollInterval: 1 * time.Second,
		PushInterval: 1 * time.Second,
		RateLimit:    10,
	}
	agent := NewAgent(config, logger, []pollers.Poller{
		pollers.NewDefaultPoller(storage.NewMemStorage()),
		pollers.NewGopsutilPoller(storage.NewMemStorage()),
	})

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*1)
	defer cancel()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := agent.Start(ctx); err != nil {
			b.Errorf("Start failed: %v", err)
		}
	}
}

func BenchmarkAgent_Run_Milisecond_1000Push(b *testing.B) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/ping" {
			w.WriteHeader(http.StatusOK)
			return
		}
		if r.URL.Path == "/updates/" {
			w.WriteHeader(http.StatusOK)
			if _, err := w.Write([]byte(`{"status": "success"}`)); err != nil {
				log.Println("write error:", err)
			}
			return
		}
		http.NotFound(w, r)
	}))
	defer mockServer.Close()

	logger := zap.NewNop()
	config := Config{
		ServerURL:    mockServer.URL,
		Key:          "test-key",
		PollInterval: 1 * time.Millisecond,
		PushInterval: 1 * time.Millisecond,
		RateLimit:    10,
	}
	agent := NewAgent(config, logger, []pollers.Poller{
		pollers.NewDefaultPoller(storage.NewMemStorage()),
		pollers.NewGopsutilPoller(storage.NewMemStorage()),
	})

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*1)
	defer cancel()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := agent.Start(ctx); err != nil {
			b.Errorf("Start failed: %v", err)
		}
	}
}
