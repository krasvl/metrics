package main

import (
	"flag"
	"os"

	"metrics/internal/server"
	"metrics/internal/storage"
)

func main() {
	addr := flag.String("a", "localhost:8080", "address")

	flag.Parse()

	addrenv := os.Getenv("ADDRESS")
	if addrenv != "" {
		addr = &addrenv
	}

	memStorage := storage.NewMemStorage()
	srv := server.NewServer(*addr, memStorage)

	srv.Start()
}
