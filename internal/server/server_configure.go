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
	defer func() {
		if err := logger.Sync(); err != nil {
			logger.Error("Cant sync logger", zap.Error(err))
		}
	}()

	return NewServer(*addr, memStorage, logger)
}
