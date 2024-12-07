package server

import (
	"flag"
	"os"

	"metrics/internal/storage"
)

func GetConfiguredServer(addrDefault string) *Server {
	addr := flag.String("a", addrDefault, "address")

	flag.Parse()

	if value, exist := os.LookupEnv("ADDRESS"); exist && value != "" {
		addr = &value
	}

	memStorage := storage.NewMemStorage()
	return NewServer(*addr, memStorage)
}
