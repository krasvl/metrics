package main

import (
	"context"
	"log"
	"metrics/internal/agent"
)

func Run(ctx context.Context) error {
	addrDefault := "localhost:8080"
	pushDefault := 10_000
	pollDefault := 2_000
	keyDefault := ""
	rateLimitDefault := 1000

	agnt, err := agent.GetConfiguredAgent(addrDefault, pushDefault, pollDefault, keyDefault, rateLimitDefault)
	if err != nil {
		return err
	}

	return agnt.Start(ctx)
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := Run(ctx); err != nil {
		log.Printf("Agent error: %v", err)
	}
}
