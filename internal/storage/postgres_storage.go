package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"go.uber.org/zap"
)

type PostgresStorage struct {
	MemStorage
	logger           *zap.Logger
	db               *sql.DB
	connectionString string
}

func NewPosgresStorage(connectionString string, logger *zap.Logger) (*PostgresStorage, error) {
	storage := &PostgresStorage{
		MemStorage:       *NewMemStorage(),
		connectionString: connectionString,
		logger:           logger,
	}

	db, err := sql.Open("pgx", connectionString)
	if err != nil {
		return nil, fmt.Errorf("cant open db connection: %s, err: %w", connectionString, err)
	}
	storage.db = db

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	if err := storage.Ping(ctx); err != nil {
		return nil, err
	}

	return storage, nil
}

func (s *PostgresStorage) Ping(ctx context.Context) error {
	if err := s.db.PingContext(ctx); err != nil {
		s.logger.Error("ping db fail", zap.Error(err))
		return fmt.Errorf("ping db fail: %w", err)
	}
	return nil
}

func (s *PostgresStorage) Close() error {
	if err := s.db.Close(); err != nil {
		s.logger.Error("cant close db connection", zap.Error(err))
		return fmt.Errorf("cant close db connection: %w", err)
	}
	return nil
}
