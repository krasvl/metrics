package main

import (
	"log"

	"metrics/internal/agent"
)

func main() {
	addrDefault := "localhost:8080"
	pushDefault := 10
	pollDefault := 2
	agnt := agent.GetConfiguredAgent(addrDefault, pushDefault, pollDefault)

	if err := agnt.Start(); err != nil {
		log.Fatalf("Agent error: %v", err)
	}
}
