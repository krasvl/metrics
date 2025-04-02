package server

import (
	"flag"
	"fmt"
	"os"
	"strconv"

	"go.uber.org/zap"

	"metrics/internal/storage"
)

// GetConfiguredServer initializes and configures a new Server instance.
// It reads configuration from flags and environment variables, sets up storage, and returns the configured server.
func GetConfiguredServer(
	addrDefault string,
	intervalDefault int,
	fileDefault string,
	restoreDefault bool,
	databaseDefault string,
	keyDefault string,
) (*Server, error) {
	fs := flag.NewFlagSet("server", flag.ContinueOnError)

	addr := fs.String("a", addrDefault, "address")
	interval := fs.Int("i", intervalDefault, "store interval")
	file := fs.String("f", fileDefault, "file storage path")
	restore := fs.Bool("r", restoreDefault, "restore from file")
	database := fs.String("d", databaseDefault, "database DSN")
	key := fs.String("k", keyDefault, "encryption key")

	if err := fs.Parse([]string{}); err != nil {
		return nil, fmt.Errorf("failed to parse empty flags: %w", err)
	}

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
	if value, ok := os.LookupEnv("KEY"); ok && value != "" {
		key = &value
	}

	logger, err := zap.NewProduction()
	if err != nil {
		return nil, fmt.Errorf("cant create logger: %w", err)
	}

	config := &Config{
		Address:         *addr,
		StoreInterval:   *interval,
		FileStoragePath: *file,
		Restore:         *restore,
		DatabaseDSN:     *database,
		Key:             *key,
	}

	var serverStorage storage.MetricsStorage = nil

	if config.DatabaseDSN != "" {
		db, err := storage.NewDB(*database)
		if err != nil {
			return nil, fmt.Errorf("cant open database: %w", err)
		}
		serverStorage, err = storage.NewPosgresStorage(db, logger)
		if err != nil {
			logger.Warn("cant create postgres storage", zap.Error(err))
		} else {
			logger.Info("use postgres storage", zap.Error(err))
		}
	}

	if config.FileStoragePath != "" && serverStorage == nil {
		serverStorage, err = storage.NewFileStorage(config.FileStoragePath, config.StoreInterval, config.Restore, logger)
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

	server := NewServer(serverStorage, logger, config)

	logger.Info("server started:",
		zap.String("addr", config.Address),
		zap.Int("interval", config.StoreInterval),
		zap.String("file", config.FileStoragePath),
		zap.Bool("restore", config.Restore),
		zap.String("database", config.DatabaseDSN),
	)

	return server, nil
}
