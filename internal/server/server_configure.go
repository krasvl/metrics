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
	databaseDefault string,
) (*Server, error) {
	addr := flag.String("a", addrDefault, "address")
	interval := flag.Int("i", intervalDefault, "interval")
	file := flag.String("f", fileDefault, "file")
	restore := flag.Bool("r", restoreDefault, "restore")
	database := flag.String("d", databaseDefault, "database-dsn")

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
	if value, ok := os.LookupEnv("DATABASE_DSN"); ok && value != "" {
		database = &value
	}

	logger, err := zap.NewProduction()
	if err != nil {
		return nil, fmt.Errorf("cant create logger: %w", err)
	}

	var serverStorage storage.MetricsStorage = nil

	if *database != "" {
		serverStorage, err = storage.NewPosgresStorage(*database, logger)
		if err != nil {
			logger.Warn("cant create postgres storage", zap.Error(err))
		} else {
			logger.Info("use postgres storage", zap.Error(err))
		}
	}

	if *file != "" && serverStorage == nil {
		serverStorage, err = storage.NewFileStorage(*file, *interval, *restore, logger)
		if err != nil {
			logger.Warn("cant create file storage", zap.Error(err))
		} else {
			logger.Info("use file storage", zap.Error(err))
		}
	}

	if serverStorage == nil {
		serverStorage = storage.NewMemStorage()
		logger.Info("use mem storage", zap.Error(err))
	}

	server := NewServer(*addr, serverStorage, logger)

	logger.Info("server started:",
		zap.String("addr", *addr),
		zap.Int("interval", *interval),
		zap.String("file", *file),
		zap.Bool("restore", *restore),
		zap.String("database", *database),
	)

	return server, nil
}
