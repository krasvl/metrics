package main

import (
	"metrics/internal/agent"
	"metrics/internal/storage"

	"log"
	"time"
)

func main() {
	s := storage.NewMemStorage()
	a := agent.NewAgent("http://localhost:8080", s, 2*time.Second, 10*time.Second)

	if err := a.Start(); err != nil {
		log.Fatalf("Agent error: %v", err)
	}
}
