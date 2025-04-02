package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"metrics/internal/utils"
	"os"
	"time"

	"go.uber.org/zap"
)

type FileMetric struct {
	Value       *float64 `json:"value"`
	Delta       *int64   `json:"delta"`
	StringValue string   `json:"string_value"`
	ID          string   `json:"id"`
	MType       string   `json:"mtype"`
	Hash        string   `json:"hash"`
}

type FileStorage struct {
	MemStorage
	logger       *zap.Logger
	file         string
	pushInterval int
}

func NewFileStorage(file string, pushInterval int, restore bool, logger *zap.Logger) (*FileStorage, error) {
	storage := &FileStorage{
		MemStorage:   *NewMemStorage(),
		file:         file,
		pushInterval: pushInterval,
		logger:       logger,
	}

	if restore {
		if err := storage.loadFromFile(); err != nil {
			return nil, fmt.Errorf("cant load file: %w", err)
		}
	}

	if pushInterval > 0 {
		ticker := time.NewTicker(time.Duration(pushInterval) * time.Millisecond)
		go func() {
			for range ticker.C {
				if err := storage.saveToFile(context.Background()); err != nil {
					logger.Error("cant save file", zap.String("file", file), zap.Error(err))
				}
			}
		}()
	}

	return storage, nil
}

func (fs *FileStorage) SetGauge(ctx context.Context, name string, value Gauge) error {
	return fs.withRetry(ctx, func() error { return fs.MemStorage.SetGauge(ctx, name, value) })
}

func (fs *FileStorage) SetGauges(ctx context.Context, values map[string]Gauge) error {
	return fs.withRetry(ctx, func() error { return fs.MemStorage.SetGauges(ctx, values) })
}

func (fs *FileStorage) ClearGauges(ctx context.Context) error {
	return fs.withRetry(ctx, func() error { return fs.MemStorage.ClearGauges(ctx) })
}

func (fs *FileStorage) SetCounter(ctx context.Context, name string, value Counter) error {
	return fs.withRetry(ctx, func() error { return fs.MemStorage.SetCounter(ctx, name, value) })
}

func (fs *FileStorage) SetCounters(ctx context.Context, values map[string]Counter) error {
	return fs.withRetry(ctx, func() error { return fs.MemStorage.SetCounters(ctx, values) })
}

func (fs *FileStorage) ClearCounters(ctx context.Context) error {
	return fs.withRetry(ctx, func() error { return fs.MemStorage.ClearCounters(ctx) })
}

func (fs *FileStorage) withRetry(ctx context.Context, op func() error) error {
	if err := op(); err != nil {
		return err
	}
	if fs.pushInterval == 0 {
		if err := fs.saveToFile(ctx); err != nil {
			fs.logger.Error("cant save file", zap.String("file", fs.file), zap.Error(err))
			return err
		}
	}
	return nil
}

func (fs *FileStorage) loadFromFile() error {
	f, err := utils.WithFileRetry(func() (*os.File, error) {
		return os.OpenFile(fs.file, os.O_RDONLY|os.O_CREATE, os.ModeAppend)
	})
	if err != nil {
		return fmt.Errorf("cant open file: %w", err)
	}
	defer func() {
		if closeErr := f.Close(); closeErr != nil {
			err = fmt.Errorf("cant close file: %w", closeErr)
		}
	}()

	data := []FileMetric{}
	if err := json.NewDecoder(f).Decode(&data); err != nil && err.Error() != "EOF" {
		return fmt.Errorf("cant decode file: %w", err)
	}

	syncTimeout := time.Minute
	ctx, cancel := context.WithTimeout(context.Background(), syncTimeout)
	defer cancel()

	for _, metric := range data {
		switch metric.MType {
		case "gauge":
			if err := fs.SetGauge(ctx, metric.ID, Gauge(*metric.Value)); err != nil {
				return err
			}
		case "counter":
			if err := fs.SetCounter(ctx, metric.ID, Counter(*metric.Delta)); err != nil {
				return err
			}
		}
	}

	return nil
}

func (fs *FileStorage) saveToFile(ctx context.Context) error {
	f, err := utils.WithFileRetry(func() (*os.File, error) {
		return os.Create(fs.file)
	})
	if err != nil {
		return fmt.Errorf("cant create file: %w", err)
	}
	defer func() {
		if closeErr := f.Close(); closeErr != nil {
			err = fmt.Errorf("cant close file: %w", closeErr)
		}
	}()

	data := []FileMetric{}

	gauges, _ := fs.GetGauges(ctx)
	for id, value := range gauges {
		floatValue := float64(value)
		metric := FileMetric{
			ID:    id,
			MType: "gauge",
			Value: &floatValue,
		}
		data = append(data, metric)
	}

	counters, _ := fs.GetCounters(ctx)
	for id, delta := range counters {
		intDelta := int64(delta)
		metric := FileMetric{
			ID:    id,
			MType: "counter",
			Delta: &intDelta,
		}
		data = append(data, metric)
	}

	if err := json.NewEncoder(f).Encode(data); err != nil {
		return fmt.Errorf("cant encode data: %w", err)
	}

	return nil
}
