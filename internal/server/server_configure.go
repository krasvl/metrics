package server

import (
	"flag"
	"os"

	"go.uber.org/zap"

	"metrics/internal/storage"
)

func GetConfiguredServer(addrDefault string) *Server {
	addr := flag.String("a", addrDefault, "address")

	flag.Parse()

	if value, ok := os.LookupEnv("ADDRESS"); ok && value != "" {
		addr = &value
	}

	memStorage := storage.NewMemStorage()

	logger, _ := zap.NewProduction()
	defer logger.Sync()

	return NewServer(*addr, memStorage, logger)
}
