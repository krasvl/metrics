package main

import (
	"log"
	"metrics/internal/server"
)

func main() {
	addrDefault := "localhost:8080"
	srv := server.GetConfiguredServer(addrDefault)

	if err := srv.Start(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
