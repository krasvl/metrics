package agent

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/go-resty/resty/v2"
	"go.uber.org/zap"

	"metrics/internal/pollers"
	"metrics/internal/utils"
)

// Config holds the configuration parameters for the Agent.
type Config struct {
	ServerURL    string
	Key          string
	PollInterval time.Duration
	PushInterval time.Duration
	RateLimit    int
}

// Agent represents a metrics collection and reporting agent.
type Agent struct {
	client  *resty.Client
	logger  *zap.Logger
	pollers []pollers.Poller
	config  Config
}

// NewAgent creates a new instance of Agent.
func NewAgent(config Config, logger *zap.Logger, pollerList []pollers.Poller) *Agent {
	return &Agent{
		config:  config,
		client:  resty.New(),
		logger:  logger,
		pollers: pollerList,
	}
}

// Start begins the metrics collection and reporting process.
func (a *Agent) Start(ctx context.Context) error {
	pollTicker := time.NewTicker(a.config.PollInterval)
	reportTicker := time.NewTicker(a.config.PushInterval)

	if err := a.testPing(); err != nil {
		return fmt.Errorf("cant ping server: %w", err)
	}
	a.logger.Info("ping server successfully")

	jobs := make(chan []pollers.Metric, a.config.RateLimit)
	done := make(chan struct{}, a.config.RateLimit)

	var wg sync.WaitGroup

	for range a.config.RateLimit {
		wg.Add(1)
		go func() {
			defer wg.Done()
			a.pushWorker(jobs, done)
		}()
	}

	for {
		select {
		case <-ctx.Done():
			pollTicker.Stop()
			reportTicker.Stop()

			close(jobs)
			wg.Wait()
			close(done)

			return nil

		case <-pollTicker.C:
			for _, poller := range a.pollers {
				go func(p pollers.Poller) {
					if err := p.Poll(); err != nil {
						a.logger.Error("cant poll metrics", zap.Error(err))
					}
				}(poller)
			}

		case <-reportTicker.C:
			var metrics []pollers.Metric

			mu := sync.Mutex{}
			grp := sync.WaitGroup{}
			grp.Add(len(a.pollers))

			for _, poller := range a.pollers {
				go func(p pollers.Poller) {
					defer grp.Done()
					pollerMetrics, err := p.GetMetrics(ctx)
					if err != nil {
						a.logger.Error("cant get metrics", zap.Error(err))
					}
					mu.Lock()
					metrics = append(metrics, pollerMetrics...)
					mu.Unlock()
				}(poller)
			}
			grp.Wait()

			if len(metrics) == 0 {
				a.logger.Warn("no metrics to send")
				continue
			}

			select {
			case jobs <- metrics:
			default:
				a.logger.Warn("cant push metrics, workers busy")
			}

		case <-done:
			for _, poller := range a.pollers {
				if err := poller.ResetMetrics(ctx); err != nil {
					a.logger.Error("cant reset metrics", zap.Error(err))
				}
			}
		}
	}
}

func (a *Agent) testPing() error {
	resp, err := utils.WithRestyRetry(func() (*resty.Response, error) {
		return a.client.R().Get(a.config.ServerURL + "/ping")
	})

	if err != nil {
		return fmt.Errorf("cant ping server: %w", err)
	}
	if resp.StatusCode() != http.StatusOK {
		return fmt.Errorf("bad status code: %d", resp.StatusCode())
	}
	return nil
}

func (a *Agent) pushWorker(jobs <-chan []pollers.Metric, done chan<- struct{}) {
	for metrics := range jobs {
		if err := a.pushMetrics(metrics); err != nil {
			a.logger.Error("cant push metrics", zap.Error(err))
			continue
		}
		a.logger.Info("metrics pushed successfully")
		done <- struct{}{}
	}
}

func (a *Agent) pushMetrics(metrics []pollers.Metric) error {
	var compressed bytes.Buffer
	writer := gzip.NewWriter(&compressed)
	if err := json.NewEncoder(writer).Encode(metrics); err != nil {
		return fmt.Errorf("cant gzip metrics: %w", err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("cant close gzip writer: %w", err)
	}

	hash := utils.GetHash(a.config.Key, compressed.Bytes())

	resp, err := utils.WithRestyRetry(func() (*resty.Response, error) {
		request := a.client.R().
			SetHeader("Content-Type", "application/json").
			SetHeader("Content-Encoding", "gzip").
			SetHeader("HashSHA256", hash).
			SetBody(compressed.Bytes())

		a.logger.Debug("Sending metrics",
			zap.String("url", a.config.ServerURL+"/updates/"),
			zap.Any("headers", request.Header),
			zap.ByteString("body", compressed.Bytes()),
		)

		return request.Post(a.config.ServerURL + "/updates/")
	})

	if err != nil {
		a.logger.Error("Failed to send metrics", zap.Error(err))
		return fmt.Errorf("cant send metrics: %w", err)
	}
	if resp.StatusCode() != http.StatusOK {
		a.logger.Error("Unexpected response from server",
			zap.Int("status_code", resp.StatusCode()),
			zap.String("body", resp.String()),
		)
		return fmt.Errorf("bad status code: %d", resp.StatusCode())
	}
	return nil
}
