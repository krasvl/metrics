package agent

import (
	"flag"
	"os"
	"strconv"
	"strings"
	"time"

	"metrics/internal/storage"
)

func GetConfiguredAgent(addrDefault string, pushDefault int, pollDefault int) *Agent {
	addr := flag.String("a", addrDefault, "address")
	pushInterval := flag.Int("r", pushDefault, "push interval")
	pollInterval := flag.Int("p", pollDefault, "poll interval")

	flag.Parse()

	if value, exist := os.LookupEnv("ADDRESS"); exist && value != "" {
		addr = &value
	}

	if value, exist := os.LookupEnv("REPORT_INTERVAL"); exist && value != "" {
		parsed, err := strconv.ParseInt(value, 10, 32)
		if err == nil && parsed > 0 {
			valueint := int(parsed)
			pushInterval = &valueint
		}
	}

	if value, exist := os.LookupEnv("POLL_INTERVAL"); exist && value != "" {
		parsed, err := strconv.ParseInt(value, 10, 32)
		if err == nil && parsed > 0 {
			valueint := int(parsed)
			pollInterval = &valueint
		}
	}

	if !strings.HasPrefix(*addr, "http://") && !strings.HasPrefix(*addr, "https://") {
		*addr = "http://" + *addr
	}

	s := storage.NewMemStorage()
	return NewAgent(*addr, s, time.Duration(*pollInterval)*time.Second, time.Duration(*pushInterval)*time.Second)
}
