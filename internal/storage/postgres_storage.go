package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx"
	_ "github.com/jackc/pgx/v5/stdlib"

	"go.uber.org/zap"
)

type PostgresStorage struct {
	logger *zap.Logger
	db     *sql.DB
}

func NewPosgresStorage(db *sql.DB, logger *zap.Logger) (*PostgresStorage, error) {
	storage := &PostgresStorage{
		db:     db,
		logger: logger,
	}

	return storage, nil
}

func (s *PostgresStorage) Ping(ctx context.Context) error {
	err := s.withRetry(func() error {
		return s.db.PingContext(ctx)
	})
	if err != nil {
		return fmt.Errorf("ping db fail: %w", err)
	}
	return nil
}

func (s *PostgresStorage) GetGauges(ctx context.Context) (map[string]Gauge, error) {
	var rows *sql.Rows
	var err error
	err = s.withRetry(func() error {
		rows, err = s.db.QueryContext(ctx, "SELECT name, value FROM gauges")
		if err != nil {
			return fmt.Errorf("cant query gauges: %w", err)
		}
		if rows.Err() != nil {
			return fmt.Errorf("rows error: %w", rows.Err())
		}
		defer func() {
			if closeErr := rows.Close(); closeErr != nil {
				s.logger.Error("Error closing rows for counters", zap.Error(closeErr))
			}
		}()
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("cant query gauges: %w", err)
	}

	result := make(map[string]Gauge)
	for rows.Next() {
		var name string
		var value Gauge
		if err := rows.Scan(&name, &value); err != nil {
			s.logger.Error("Error scanning gauge row", zap.Error(err))
			continue
		}
		result[name] = value
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return result, nil
}

func (s *PostgresStorage) GetGauge(ctx context.Context, name string) (Gauge, bool, error) {
	var value Gauge
	err := s.withRetry(func() error {
		return s.db.QueryRowContext(ctx, "SELECT value FROM gauges WHERE name = $1", name).Scan(&value)
	})
	if errors.Is(err, sql.ErrNoRows) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, fmt.Errorf("cant query gauge: %w", err)
	}
	return value, true, nil
}

func (s *PostgresStorage) SetGauge(ctx context.Context, name string, value Gauge) error {
	err := s.withRetry(func() error {
		_, err := s.db.ExecContext(
			ctx,
			"INSERT INTO gauges (name, value) VALUES ($1, $2) ON CONFLICT (name) DO UPDATE SET value = $2",
			name,
			value,
		)
		return err
	})
	if err != nil {
		return fmt.Errorf("cant set gauge: %w", err)
	}
	return nil
}

func (s *PostgresStorage) SetGauges(ctx context.Context, values map[string]Gauge) error {
	keys := make([]string, 0, len(values))
	for name := range values {
		keys = append(keys, name)
	}
	sort.Strings(keys)

	valueStrings := make([]string, 0, len(keys))
	valueArgs := make([]interface{}, 0, len(keys)*2)

	i := 1
	for _, name := range keys {
		value := values[name]
		valueStrings = append(valueStrings, fmt.Sprintf("($%d, $%d)", i, i+1))
		valueArgs = append(valueArgs, name, value)
		i += 2
	}

	stmt := fmt.Sprintf(`
		INSERT INTO gauges (name, value) 
		VALUES %s
		ON CONFLICT (name) DO UPDATE SET value = EXCLUDED.value
	`, strings.Join(valueStrings, ","))

	err := s.withRetry(func() error {
		_, err := s.db.ExecContext(ctx, stmt, valueArgs...)
		return err
	})

	if err != nil {
		return fmt.Errorf("cant set gauges: %w", err)
	}

	return nil
}

func (s *PostgresStorage) ClearGauges(ctx context.Context) error {
	err := s.withRetry(func() error {
		_, err := s.db.ExecContext(ctx, "DELETE FROM gauges")
		return err
	})
	if err != nil {
		return fmt.Errorf("cant clear all gauges: %w", err)
	}
	return nil
}

func (s *PostgresStorage) GetCounters(ctx context.Context) (map[string]Counter, error) {
	var rows *sql.Rows
	var err error
	err = s.withRetry(func() error {
		rows, err = s.db.QueryContext(ctx, "SELECT name, value FROM counters")
		if err != nil {
			return fmt.Errorf("cant query counters: %w", err)
		}
		defer func() {
			if closeErr := rows.Close(); closeErr != nil {
				s.logger.Error("Error closing rows for counters", zap.Error(closeErr))
			}
		}()
		if rows.Err() != nil {
			return fmt.Errorf("rows error: %w", rows.Err())
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("cant query counters: %w", err)
	}

	result := make(map[string]Counter)
	for rows.Next() {
		var name string
		var value Counter
		if err := rows.Scan(&name, &value); err != nil {
			s.logger.Error("Error scanning counter row", zap.Error(err))
			continue
		}
		result[name] = value
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return result, nil
}

func (s *PostgresStorage) GetCounter(ctx context.Context, name string) (Counter, bool, error) {
	var value Counter
	err := s.withRetry(func() error {
		return s.db.QueryRowContext(ctx, "SELECT value FROM counters WHERE name = $1", name).Scan(&value)
	})
	if errors.Is(err, sql.ErrNoRows) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, fmt.Errorf("cant get counter: %w", err)
	}
	return value, true, nil
}

func (s *PostgresStorage) SetCounter(ctx context.Context, name string, value Counter) error {
	err := s.withRetry(func() error {
		_, err := s.db.ExecContext(
			ctx,
			"INSERT INTO counters (name, value) VALUES ($1, $2) ON CONFLICT (name) DO UPDATE SET value = counters.value + $2",
			name,
			value,
		)
		return err
	})
	if err != nil {
		return fmt.Errorf("cant set counter: %w", err)
	}
	return nil
}

func (s *PostgresStorage) SetCounters(ctx context.Context, values map[string]Counter) error {
	keys := make([]string, 0, len(values))
	for name := range values {
		keys = append(keys, name)
	}
	sort.Strings(keys)

	valueStrings := make([]string, 0, len(keys))
	valueArgs := make([]interface{}, 0, len(keys)*2)

	i := 1
	for _, name := range keys {
		value := values[name]
		valueStrings = append(valueStrings, fmt.Sprintf("($%d, $%d)", i, i+1))
		valueArgs = append(valueArgs, name, value)
		i += 2
	}

	stmt := fmt.Sprintf(`
		INSERT INTO counters (name, value) 
		VALUES %s
		ON CONFLICT (name) DO UPDATE SET value = counters.value + EXCLUDED.value
	`, strings.Join(valueStrings, ","))

	err := s.withRetry(func() error {
		_, err := s.db.ExecContext(ctx, stmt, valueArgs...)
		return err
	})

	if err != nil {
		return fmt.Errorf("cant set counters: %w", err)
	}

	return nil
}

func (s *PostgresStorage) ClearCounters(ctx context.Context) error {
	err := s.withRetry(func() error {
		_, err := s.db.ExecContext(ctx, "DELETE FROM counters")
		return err
	})
	if err != nil {
		return fmt.Errorf("cant clear all counters: %w", err)
	}
	return nil
}

func (s *PostgresStorage) withRetry(exec func() error) error {
	var err error
	for _, delay := range []int{0, 1, 3, 5} {
		time.Sleep(time.Duration(delay) * time.Second)
		err = exec()

		if err != nil {
			if strings.Contains(err.Error(), "deadlock detected") {
				s.logger.Warn("Deadlock detected, retrying", zap.Error(err))
				continue
			}
			if errors.Is(err, pgx.ErrConnBusy) || !errors.Is(err, pgx.ErrDeadConn) {
				s.logger.Warn("Connection fail, retry", zap.Error(err))
				continue
			}
		}

		return err
	}
	return err
}
