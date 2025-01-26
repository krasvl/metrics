package main

import (
	"log"

	"metrics/internal/agent"
)

func main() {
	addrDefault := "localhost:8080"
	pushDefault := 10
	pollDefault := 2
	key := ""
	agnt, err := agent.GetConfiguredAgent(addrDefault, pushDefault, pollDefault, key)

	if err != nil {
		log.Fatalf("Agent configure error: %v", err)
	}

	if err := agnt.Start(); err != nil {
		log.Fatalf("Agent error: %v", err)
	}
}
