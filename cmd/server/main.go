package main

import (
	"metrics/internal/server"
	"metrics/internal/storage"
)

func main() {
	memStorage := storage.NewMemStorage()
	srv := server.NewServer(memStorage)

	srv.Start()
}
