package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"go.uber.org/zap"
)

type FileMetric struct {
	Delta *Counter `json:"delta,omitempty"`
	Value *Gauge   `json:"value,omitempty"`
	ID    string   `json:"id"`
	MType string   `json:"type"`
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

	if !restore {
		return storage, nil
	}

	f, err := os.OpenFile(file, os.O_RDONLY|os.O_CREATE, os.ModeAppend)
	if err != nil {
		return nil, fmt.Errorf("cant open file: %w", err)
	}
	defer func() {
		if closeErr := f.Close(); closeErr != nil {
			err = fmt.Errorf("cant close file: %w", closeErr)
		}
	}()

	data := []FileMetric{}
	if err := json.NewDecoder(f).Decode(&data); err != nil && err.Error() != "EOF" {
		return nil, fmt.Errorf("cant decode file: %w", err)
	}

	syncTimeout := time.Minute
	ctx, cancel := context.WithTimeout(context.Background(), syncTimeout)
	defer cancel()

	for _, metric := range data {
		switch metric.MType {
		case "gauge":
			if metric.Value != nil {
				_ = storage.SetGauge(ctx, metric.ID, *metric.Value)
			}
		case "counter":
			if metric.Delta != nil {
				_ = storage.SetCounter(ctx, metric.ID, *metric.Delta)
			}
		}
	}

	if pushInterval > 0 {
		ticker := time.NewTicker(time.Duration(pushInterval) * time.Second)
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

func (fs *FileStorage) saveToFile(ctx context.Context) error {
	f, err := os.Create(fs.file)
	if err != nil {
		return fmt.Errorf("cant create file: %w", err)
	}
	defer func() {
		if closeErr := f.Close(); closeErr != nil {
			err = fmt.Errorf("cant close file: %w", closeErr)
		}
	}()

	data := []FileMetric{}

	gauges, _ := fs.GetAllGauges(ctx)
	for id, value := range gauges {
		metric := FileMetric{
			ID:    id,
			MType: "gauge",
			Value: &value,
		}
		data = append(data, metric)
	}

	counters, _ := fs.GetAllCounters(ctx)
	for id, delta := range counters {
		metric := FileMetric{
			ID:    id,
			MType: "counter",
			Delta: &delta,
		}
		data = append(data, metric)
	}

	if err := json.NewEncoder(f).Encode(data); err != nil {
		return fmt.Errorf("cant encode data: %w", err)
	}

	return nil
}

func (fs *FileStorage) SetGauge(ctx context.Context, name string, value Gauge) error {
	_ = fs.MemStorage.SetGauge(ctx, name, value)
	if fs.pushInterval == 0 {
		if err := fs.saveToFile(ctx); err != nil {
			fs.logger.Error("cant save file", zap.String("file", fs.file), zap.Error(err))
			return err
		}
	}
	return nil
}

func (fs *FileStorage) ClearGauge(ctx context.Context, name string) error {
	_ = fs.MemStorage.ClearGauge(ctx, name)
	if fs.pushInterval == 0 {
		if err := fs.saveToFile(ctx); err != nil {
			fs.logger.Error("cant save file", zap.String("file", fs.file), zap.Error(err))
			return err
		}
	}
	return nil
}

func (fs *FileStorage) ClearGauges(ctx context.Context) error {
	_ = fs.MemStorage.ClearGauges(ctx)
	if fs.pushInterval == 0 {
		if err := fs.saveToFile(ctx); err != nil {
			fs.logger.Error("cant save file", zap.String("file", fs.file), zap.Error(err))
			return err
		}
	}
	return nil
}

func (fs *FileStorage) SetCounter(ctx context.Context, name string, value Counter) error {
	_ = fs.MemStorage.SetCounter(ctx, name, value)
	if fs.pushInterval == 0 {
		if err := fs.saveToFile(ctx); err != nil {
			fs.logger.Error("cant save file", zap.String("file", fs.file), zap.Error(err))
			return err
		}
	}
	return nil
}

func (fs *FileStorage) ClearCounter(ctx context.Context, name string) error {
	_ = fs.MemStorage.ClearCounter(ctx, name)
	if fs.pushInterval == 0 {
		if err := fs.saveToFile(ctx); err != nil {
			fs.logger.Error("cant save file", zap.String("file", fs.file), zap.Error(err))
			return err
		}
	}
	return nil
}

func (fs *FileStorage) ClearCounters(ctx context.Context) error {
	_ = fs.MemStorage.ClearCounters(ctx)
	if fs.pushInterval == 0 {
		if err := fs.saveToFile(ctx); err != nil {
			fs.logger.Error("cant save file", zap.String("file", fs.file), zap.Error(err))
			return err
		}
	}
	return nil
}
