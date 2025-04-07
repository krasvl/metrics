package main

import (
	"context"
	"log"
	"metrics/internal/server"
)

func Run(ctx context.Context) error {
	addrDefault := "localhost:8080"
	intervalDefault := 300_000
	fileDefault := "./store"
	restoreDefault := true
	databaseDefault := ""
	key := ""

	srv, err := server.GetConfiguredServer(addrDefault, intervalDefault, fileDefault, restoreDefault, databaseDefault, key)
	if err != nil {
		return err
	}

	return srv.Start(ctx)
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := Run(ctx); err != nil {
		log.Printf("Server error: %v", err)
	}
}
