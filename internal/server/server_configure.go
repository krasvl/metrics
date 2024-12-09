package server

import (
	"flag"
	"os"

	"metrics/internal/storage"
)

func GetConfiguredServer(addrDefault string) *Server {
	addr := flag.String("a", addrDefault, "address")

	flag.Parse()

	if value, ok := os.LookupEnv("ADDRESS"); ok && value != "" {
		addr = &value
	}

	memStorage := storage.NewMemStorage()
	return NewServer(*addr, memStorage)
}
