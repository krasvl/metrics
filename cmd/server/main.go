package main

import (
	"log"
	"metrics/internal/server"
)

func main() {
	addrDefault := "localhost:8080"
	intervalDefault := 300
	fileDefault := "./store"
	restoreDefault := true
	databaseDefault := ""

	srv, err := server.GetConfiguredServer(addrDefault, intervalDefault, fileDefault, restoreDefault, databaseDefault)

	if err != nil {
		log.Fatalf("Server configure error: %v", err)
	}

	if err := srv.Start(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
