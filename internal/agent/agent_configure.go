package agent

import (
	"flag"
	"fmt"
	"log"
	"metrics/internal/pollers"
	"metrics/internal/storage"
	"os"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"
)

// GetConfiguredAgent initializes and configures a new Agent instance.
// It reads configuration from flags and environment variables, sets up pollers, and returns the configured agent.
func GetConfiguredAgent(
	addrDefault string,
	pushDefault int,
	pollDefault int,
	keyDefault string,
	rateLimitDefault int,
) (*Agent, error) {
	fs := flag.NewFlagSet("agent", flag.ContinueOnError)

	addr := fs.String("a", addrDefault, "address")
	pushInterval := fs.Int("r", pushDefault, "push interval")
	pollInterval := fs.Int("p", pollDefault, "poll interval")
	key := fs.String("k", keyDefault, "key")
	rateLimit := fs.Int("l", rateLimitDefault, "rate limit")

	if err := fs.Parse([]string{}); err != nil {
		log.Printf("Error parsing flags: %v", err)
	}

	if value, ok := os.LookupEnv("ADDRESS"); ok && value != "" {
		addr = &value
	}

	if value, ok := os.LookupEnv("REPORT_INTERVAL"); ok && value != "" {
		parsed, err := strconv.ParseInt(value, 10, 32)
		if err == nil && parsed > 0 {
			valueint := int(parsed)
			pushInterval = &valueint
		}
	}

	if value, ok := os.LookupEnv("POLL_INTERVAL"); ok && value != "" {
		parsed, err := strconv.ParseInt(value, 10, 32)
		if err == nil && parsed > 0 {
			valueint := int(parsed)
			pollInterval = &valueint
		}
	}

	if value, ok := os.LookupEnv("KEY"); ok && value != "" {
		key = &value
	}

	if value, ok := os.LookupEnv("RATE_LIMIT"); ok && value != "" {
		parsed, err := strconv.Atoi(value)
		if err == nil && parsed > 0 {
			rateLimit = &parsed
		}
	}

	if !strings.HasPrefix(*addr, "http://") && !strings.HasPrefix(*addr, "https://") {
		*addr = "http://" + *addr
	}

	logger, err := zap.NewProduction()
	if err != nil {
		return nil, fmt.Errorf("cant create logger: %w", err)
	}

	config := Config{
		ServerURL:    *addr,
		Key:          *key,
		PollInterval: time.Duration(*pollInterval) * time.Millisecond,
		PushInterval: time.Duration(*pushInterval) * time.Millisecond,
		RateLimit:    *rateLimit,
	}

	agent := NewAgent(
		config,
		logger,
		[]pollers.Poller{
			pollers.NewDefaultPoller(storage.NewMemStorage()),
			pollers.NewGopsutilPoller(storage.NewMemStorage()),
		},
	)

	logger.Info("agent started:",
		zap.String("addr", config.ServerURL),
		zap.Duration("pushInterval", config.PushInterval),
		zap.Duration("pollInterval", config.PollInterval),
		zap.Int("rateLimit", config.RateLimit),
	)

	return agent, nil
}
