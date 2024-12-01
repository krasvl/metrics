package main

import (
	"flag"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"metrics/internal/agent"
	"metrics/internal/storage"
)

func main() {
	addr := flag.String("a", "localhost:8080", "address")
	pushInterval := flag.Int("r", 10, "push interval")
	pollInterval := flag.Int("p", 2, "poll interval")

	flag.Parse()

	addrenv := os.Getenv("ADDRESS")
	if addrenv != "" {
		addr = &addrenv
	}

	pushenv := os.Getenv("REPORT_INTERVAL")
	pushenvVal, err := strconv.ParseInt(pushenv, 10, 32)
	if err != nil {
		val := int(pushenvVal)
		pushInterval = &val
	}

	pollenv := os.Getenv("POLL_INTERVAL")
	pollenvVal, err := strconv.ParseInt(pollenv, 10, 32)
	if err != nil {
		val := int(pollenvVal)
		pollInterval = &val
	}

	if !strings.HasPrefix(*addr, "http://") && !strings.HasPrefix(*addr, "https://") {
		*addr = "http://" + *addr
	}

	s := storage.NewMemStorage()
	a := agent.NewAgent(*addr, s, time.Duration(*pollInterval)*time.Second, time.Duration(*pushInterval)*time.Second)

	if err := a.Start(); err != nil {
		log.Fatalf("Agent error: %v", err)
	}
}
