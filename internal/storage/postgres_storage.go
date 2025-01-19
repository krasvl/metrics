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
	logger           *zap.Logger
	db               *sql.DB
	connectionString string
}

func NewPosgresStorage(connectionString string, logger *zap.Logger) (*PostgresStorage, error) {
	storage := &PostgresStorage{
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

	if err := storage.initTables(ctx); err != nil {
		return nil, err
	}

	logger.Debug("db init success")
	return storage, nil
}

func (s *PostgresStorage) initTables(ctx context.Context) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS gauges (
			id SERIAL PRIMARY KEY,
			name TEXT NOT NULL UNIQUE,
			value DOUBLE PRECISION
		);`,
		`CREATE INDEX IF NOT EXISTS idx_gauges_name ON gauges (name);`,
		`CREATE TABLE IF NOT EXISTS counters (
			id SERIAL PRIMARY KEY,
			name TEXT NOT NULL UNIQUE,
			value BIGINT
		);`,
		`CREATE INDEX IF NOT EXISTS idx_counters_name ON counters (name);`,
	}

	for _, query := range queries {
		_, err := s.db.ExecContext(ctx, query)
		if err != nil {
			s.logger.Error("cant init tables", zap.Error(err))
			return fmt.Errorf("cant init tables: %w", err)
		}
	}

	return nil
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

func (s *PostgresStorage) GetGauges(ctx context.Context) (map[string]Gauge, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT name, value FROM gauges")
	if err != nil {
		s.logger.Error("cant query gauges", zap.Error(err))
		return nil, fmt.Errorf("cant query gauges: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			s.logger.Error("cant close rows", zap.Error(err))
		}
	}()

	result := make(map[string]Gauge)
	for rows.Next() {
		var name string
		var value Gauge
		if err := rows.Scan(&name, &value); err != nil {
			s.logger.Error("cant scan gauges", zap.Error(err))
			continue
		}
		result[name] = value
	}

	if err := rows.Err(); err != nil {
		s.logger.Error("rows iteration error", zap.Error(err))
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return result, nil
}

func (s *PostgresStorage) GetGauge(ctx context.Context, name string) (Gauge, bool, error) {
	var value Gauge
	err := s.db.QueryRowContext(ctx, "SELECT value FROM gauges WHERE name = $1", name).Scan(&value)
	if err == sql.ErrNoRows {
		return 0, false, nil
	}
	if err != nil {
		s.logger.Error("cant query gauge", zap.String("name", name), zap.Error(err))
		return 0, false, fmt.Errorf("cant query gauge: %w", err)
	}
	return value, true, nil
}

func (s *PostgresStorage) SetGauge(ctx context.Context, name string, value Gauge) error {
	_, err := s.db.ExecContext(
		ctx,
		"INSERT INTO gauges (name, value) VALUES ($1, $2) ON CONFLICT (name) DO UPDATE SET value = $2",
		name,
		value,
	)
	if err != nil {
		s.logger.Error("cant set gauge", zap.String("name", name), zap.Float64("value", float64(value)), zap.Error(err))
		return fmt.Errorf("cant set gauge: %w", err)
	}
	return nil
}

func (s *PostgresStorage) SetGauges(ctx context.Context, values map[string]Gauge) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		s.logger.Error("cant start transaction", zap.Error(err))
		return fmt.Errorf("cant start transaction: %w", err)
	}

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO gauges (name, value) 
		VALUES ($1, $2) 
		ON CONFLICT (name) DO UPDATE SET value = EXCLUDED.value
	`)
	if err != nil {
		s.logger.Error("cant prepare statement", zap.Error(err))
		_ = tx.Rollback()
		return fmt.Errorf("cant prepare statement: %w", err)
	}
	defer func() {
		if err := stmt.Close(); err != nil {
			s.logger.Error("cant close stmt", zap.Error(err))
		}
	}()

	for name, value := range values {
		if _, err := stmt.ExecContext(ctx, name, value); err != nil {
			s.logger.Error(
				"cant execute statement",
				zap.String("name", name),
				zap.Float64("value", float64(value)),
				zap.Error(err),
			)
			_ = tx.Rollback()
			return fmt.Errorf("cant execute statement: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		s.logger.Error("cant commit transaction", zap.Error(err))
		return fmt.Errorf("cant commit transaction: %w", err)
	}

	return nil
}

func (s *PostgresStorage) ClearGauge(ctx context.Context, name string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM gauges WHERE name = $1", name)
	if err != nil {
		s.logger.Error("cant clear gauge", zap.String("name", name), zap.Error(err))
		return fmt.Errorf("cant clear gauge: %w", err)
	}
	return nil
}

func (s *PostgresStorage) ClearGauges(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM gauges")
	if err != nil {
		s.logger.Error("cant clear all gauges", zap.Error(err))
		return fmt.Errorf("cant clear all gauges: %w", err)
	}
	return nil
}

func (s *PostgresStorage) GetCounters(ctx context.Context) (map[string]Counter, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT name, value FROM counters")
	if err != nil {
		s.logger.Error("cant query counters", zap.Error(err))
		return nil, fmt.Errorf("cant query counters: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			s.logger.Error("cant close rows", zap.Error(err))
		}
	}()

	result := make(map[string]Counter)
	for rows.Next() {
		var name string
		var value Counter
		if err := rows.Scan(&name, &value); err != nil {
			s.logger.Error("cant scan counter row", zap.Error(err))
			continue
		}
		result[name] = value
	}

	if err := rows.Err(); err != nil {
		s.logger.Error("rows iteration error", zap.Error(err))
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return result, nil
}

func (s *PostgresStorage) GetCounter(ctx context.Context, name string) (Counter, bool, error) {
	var value Counter
	err := s.db.QueryRowContext(ctx, "SELECT value FROM counters WHERE name = $1", name).Scan(&value)
	if err == sql.ErrNoRows {
		return 0, false, nil
	}
	if err != nil {
		s.logger.Error("cant get counter", zap.String("name", name), zap.Error(err))
		return 0, false, fmt.Errorf("cant get counter: %w", err)
	}
	return value, true, nil
}

func (s *PostgresStorage) SetCounter(ctx context.Context, name string, value Counter) error {
	_, err := s.db.ExecContext(
		ctx,
		"INSERT INTO counters (name, value) VALUES ($1, $2) ON CONFLICT (name) DO UPDATE SET value = counters.value + $2",
		name,
		value,
	)
	if err != nil {
		s.logger.Error("cant set counter", zap.String("name", name), zap.Int("value", int(value)), zap.Error(err))
		return fmt.Errorf("cant set counter: %w", err)
	}
	return nil
}

func (s *PostgresStorage) SetCounters(ctx context.Context, values map[string]Counter) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		s.logger.Error("cant start transaction", zap.Error(err))
		return fmt.Errorf("cant start transaction: %w", err)
	}

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO counters (name, value) 
		VALUES ($1, $2) 
		ON CONFLICT (name) DO UPDATE SET value = counters.value + EXCLUDED.value
	`)
	if err != nil {
		s.logger.Error("cant prepare statement", zap.Error(err))
		_ = tx.Rollback()
		return fmt.Errorf("cant prepare statement: %w", err)
	}
	defer func() {
		if err := stmt.Close(); err != nil {
			s.logger.Error("cant close stmt", zap.Error(err))
		}
	}()

	for name, value := range values {
		if _, err := stmt.ExecContext(ctx, name, value); err != nil {
			s.logger.Error(
				"cant execute statement",
				zap.String("name", name),
				zap.Int("value", int(value)),
				zap.Error(err),
			)
			_ = tx.Rollback()
			return fmt.Errorf("cant execute statement: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		s.logger.Error("cant commit transaction", zap.Error(err))
		return fmt.Errorf("cant commit transaction: %w", err)
	}

	return nil
}

func (s *PostgresStorage) ClearCounter(ctx context.Context, name string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM counters WHERE name = $1", name)
	if err != nil {
		s.logger.Error("cant clear counter", zap.String("name", name), zap.Error(err))
		return fmt.Errorf("cant clear counter: %w", err)
	}
	return nil
}

func (s *PostgresStorage) ClearCounters(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM counters")
	if err != nil {
		s.logger.Error("cant clear all counters", zap.Error(err))
		return fmt.Errorf("cant clear all counters: %w", err)
	}
	return nil
}
