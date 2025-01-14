package server

import (
	"flag"
	"fmt"
	"os"
	"strconv"

	"go.uber.org/zap"

	"metrics/internal/storage"
)

func GetConfiguredServer(
	addrDefault string,
	intervalDefault int,
	fileDefault string,
	restoreDefault bool,
) (*Server, error) {
	addr := flag.String("a", addrDefault, "address")
	interval := flag.Int("i", intervalDefault, "interval")
	file := flag.String("f", fileDefault, "file")
	restore := flag.Bool("r", restoreDefault, "restore")

	flag.Parse()

	if value, ok := os.LookupEnv("ADDRESS"); ok && value != "" {
		addr = &value
	}
	if value, ok := os.LookupEnv("STORE_INTERVAL"); ok && value != "" {
		parsed, err := strconv.ParseInt(value, 10, 32)
		if err == nil && parsed >= 0 {
			valueint := int(parsed)
			interval = &valueint
		}
	}
	if value, ok := os.LookupEnv("FILE_STORAGE_PATH"); ok && value != "" {
		file = &value
	}
	if value, ok := os.LookupEnv("RESTORE"); ok && value != "" {
		parsed, err := strconv.ParseBool(value)
		if err == nil {
			restore = &parsed
		}
	}

	logger, _ := zap.NewProduction()
	defer func() {
		if err := logger.Sync(); err != nil {
			logger.Error("Cant sync logger", zap.Error(err))
		}
	}()

	fileStorage, err := storage.NewFileStorage(*file, *interval, *restore, logger)
	if err != nil {
		return nil, fmt.Errorf("cant create fileStorage: %w", err)
	}

	return NewServer(*addr, fileStorage, logger), nil
}
