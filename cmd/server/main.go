package main

import (
	"flag"

	"metrics/internal/server"
	"metrics/internal/storage"
)

func main() {
	addr := flag.String("a", "localhost:8080", "address")

	flag.Parse()

	memStorage := storage.NewMemStorage()
	srv := server.NewServer(*addr, memStorage)

	srv.Start()
}
