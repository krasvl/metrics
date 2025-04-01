package agent

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	"metrics/internal/pollers"
)

type MockPoller struct {
	mock.Mock
}

func (m *MockPoller) Poll() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockPoller) GetMetrics(ctx context.Context) ([]pollers.Metric, error) {
	args := m.Called()
	metrics, ok := args.Get(0).([]pollers.Metric)
	if !ok {
		return nil, errors.New("invalid type assertion for metrics")
	}
	err := args.Error(1)
	return metrics, err
}

func (m *MockPoller) ResetMetrics(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func TestAgent_testPing(t *testing.T) {
	t.Run("successful ping", func(t *testing.T) {
		logger := zaptest.NewLogger(t)
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/ping" {
				w.WriteHeader(http.StatusOK)
			} else {
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		defer server.Close()

		agent := NewAgent(Config{ServerURL: server.URL}, logger, []pollers.Poller{})
		err := agent.testPing()
		assert.NoError(t, err)
	})

	t.Run("unsuccess ping", func(t *testing.T) {
		logger := zaptest.NewLogger(t)
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/ping" {
				w.WriteHeader(http.StatusBadRequest)
			} else {
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		defer server.Close()

		agent := NewAgent(Config{ServerURL: server.URL}, logger, []pollers.Poller{})
		err := agent.testPing()
		assert.Error(t, err)
	})
}

func TestAgent_pushMetrics(t *testing.T) {
	t.Run("successful push", func(t *testing.T) {
		logger := zaptest.NewLogger(t)
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/updates/" {
				w.WriteHeader(http.StatusOK)
			} else {
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		defer server.Close()

		metrics := []pollers.Metric{
			{ID: "test_metric", Value: float64Ptr(123)},
		}
		agent := NewAgent(Config{ServerURL: server.URL, Key: "test_key"}, logger, []pollers.Poller{})

		err := agent.pushMetrics(metrics)
		assert.NoError(t, err)
	})

	t.Run("unsuccess push", func(t *testing.T) {
		logger := zaptest.NewLogger(t)
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/updates/" {
				w.WriteHeader(http.StatusBadRequest)
			} else {
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		defer server.Close()

		metrics := []pollers.Metric{
			{ID: "test_metric", Value: float64Ptr(123)},
		}
		agent := NewAgent(Config{ServerURL: server.URL, Key: "test_key"}, logger, []pollers.Poller{})

		err := agent.pushMetrics(metrics)
		assert.Error(t, err)
	})

	t.Run("request headers", func(t *testing.T) {
		logger := zaptest.NewLogger(t)
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/updates/" {
				assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
				assert.Equal(t, "gzip", r.Header.Get("Content-Encoding"))
				assert.NotEmpty(t, r.Header.Get("HashSHA256"))
				w.WriteHeader(http.StatusOK)
			} else {
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		defer server.Close()

		metrics := []pollers.Metric{
			{ID: "test_metric", Value: float64Ptr(123)},
		}
		agent := NewAgent(Config{ServerURL: server.URL, Key: "test_key"}, logger, []pollers.Poller{})

		err := agent.pushMetrics(metrics)
		assert.NoError(t, err)
	})
}

func TestAgent(t *testing.T) {
	t.Run("base", func(t *testing.T) {
		logger := zaptest.NewLogger(t)
		mockPoller := new(MockPoller)
		mockPoller.On("Poll").Return(nil)
		mockPoller.On("GetMetrics", mock.Anything).Return([]pollers.Metric{{ID: "test_metric", Value: float64Ptr(123)}}, nil)
		mockPoller.On("ResetMetrics", mock.Anything).Return(nil)

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/updates/":
				w.WriteHeader(http.StatusOK)
			case "/ping":
				w.WriteHeader(http.StatusOK)
			default:
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		defer server.Close()

		agent := NewAgent(Config{
			ServerURL:    server.URL,
			PollInterval: 10 * time.Millisecond,
			PushInterval: 10 * time.Millisecond,
			RateLimit:    1,
		}, logger, []pollers.Poller{mockPoller})

		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
			defer cancel()
			err := agent.Start(ctx)
			assert.NoError(t, err)
		}()

		wg.Wait()

		mockPoller.AssertCalled(t, "Poll")
		mockPoller.AssertCalled(t, "GetMetrics", mock.Anything)
		mockPoller.AssertCalled(t, "ResetMetrics", mock.Anything)
	})

	t.Run("two updates", func(t *testing.T) {
		logger := zaptest.NewLogger(t)

		var receivedMetrics [][]pollers.Metric
		var mu sync.Mutex

		mockPoller := new(MockPoller)
		mockPoller.On("Poll").Return(nil)

		mockPoller.On("GetMetrics", mock.Anything).Return([]pollers.Metric{{ID: "diff", Value: float64Ptr(1)}}, nil).Once()
		mockPoller.On("GetMetrics", mock.Anything).Return([]pollers.Metric{{ID: "diff", Value: float64Ptr(2)}}, nil).Once()

		mockPoller.On("ResetMetrics", mock.Anything).Return(nil)

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/ping":
				w.WriteHeader(http.StatusOK)
			case "/updates/":
				if r.Header.Get("Content-Encoding") == "gzip" {
					gr, err := gzip.NewReader(r.Body)
					require.NoError(t, err)
					defer func() {
						err := gr.Close()
						if err != nil {
							t.Errorf("Failed to close gzip reader: %v", err)
						}
					}()

					var batch []pollers.Metric
					err = json.NewDecoder(gr).Decode(&batch)
					require.NoError(t, err)

					mu.Lock()
					receivedMetrics = append(receivedMetrics, batch)
					mu.Unlock()
				}
				w.WriteHeader(http.StatusOK)
			default:
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		defer server.Close()

		agent := NewAgent(Config{
			ServerURL:    server.URL,
			PollInterval: 20 * time.Millisecond,
			PushInterval: 20 * time.Millisecond,
			RateLimit:    1,
		}, logger, []pollers.Poller{mockPoller})

		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
			defer cancel()
			err := agent.Start(ctx)
			assert.NoError(t, err)
		}()

		wg.Wait()

		mockPoller.AssertNumberOfCalls(t, "Poll", 2)
		mockPoller.AssertNumberOfCalls(t, "GetMetrics", 2)
		mockPoller.AssertNumberOfCalls(t, "ResetMetrics", 2)

		assert.Equal(t, 2, len(receivedMetrics))
		assert.Equal(t, float64(1), *receivedMetrics[0][0].Value)
		assert.Equal(t, float64(2), *receivedMetrics[1][0].Value)
	})
}

func float64Ptr(v float64) *float64 {
	return &v
}
