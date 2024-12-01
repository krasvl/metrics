package main

import (
	"flag"
	"metrics/internal/agent"
	"metrics/internal/storage"

	"log"
	"time"
)

func main() {
	addr := flag.String("a", "localhost:8080", "address")
	pushInterval := flag.Int("r", 10, "push interval")
	pollInterval := flag.Int("p", 2, "poll interval")

	flag.Parse()

	s := storage.NewMemStorage()
	a := agent.NewAgent(*addr, s, time.Duration(*pollInterval)*time.Second, time.Duration(*pushInterval)*time.Second)

	if err := a.Start(); err != nil {
		log.Fatalf("Agent error: %v", err)
	}
}
