package agent

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"metrics/internal/storage"

	"go.uber.org/zap"
)

func GetConfiguredAgent(addrDefault string, pushDefault int, pollDefault int) (*Agent, error) {
	addr := flag.String("a", addrDefault, "address")
	pushInterval := flag.Int("r", pushDefault, "push interval")
	pollInterval := flag.Int("p", pollDefault, "poll interval")

	flag.Parse()

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

	if !strings.HasPrefix(*addr, "http://") && !strings.HasPrefix(*addr, "https://") {
		*addr = "http://" + *addr
	}

	logger, err := zap.NewProduction()
	if err != nil {
		return nil, fmt.Errorf("cant create logger: %w", err)
	}

	s := storage.NewMemStorage()
	agent := NewAgent(*addr, s, time.Duration(*pollInterval)*time.Second, time.Duration(*pushInterval)*time.Second, logger)

	logger.Info("agent started:",
		zap.String("addr", *addr),
		zap.Int("pushInterval", *pushInterval),
		zap.Int("pollInterval", *pollInterval),
	)

	return agent, nil
}
