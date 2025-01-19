package storage

import (
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

	for _, metric := range data {
		switch metric.MType {
		case "gauge":
			if metric.Value != nil {
				storage.SetGauge(metric.ID, *metric.Value)
			}
		case "counter":
			if metric.Delta != nil {
				storage.SetCounter(metric.ID, *metric.Delta)
			}
		}
	}

	if pushInterval > 0 {
		ticker := time.NewTicker(time.Duration(pushInterval) * time.Second)
		go func() {
			for range ticker.C {
				if err := storage.saveToFile(); err != nil {
					logger.Error("cant create file", zap.String("file", file), zap.Error(err))
				}
			}
		}()
	}

	return storage, nil
}

func (fs *FileStorage) saveToFile() error {
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

	for id, value := range fs.GetAllGauges() {
		metric := FileMetric{
			ID:    id,
			MType: "gauge",
			Value: &value,
		}
		data = append(data, metric)
	}

	for id, delta := range fs.GetAllCounters() {
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

func (fs *FileStorage) SetGauge(name string, value Gauge) {
	fs.MemStorage.SetGauge(name, value)
	if fs.pushInterval == 0 {
		if err := fs.saveToFile(); err != nil {
			fs.logger.Error("cant create file", zap.String("file", fs.file), zap.Error(err))
		}
	}
}

func (fs *FileStorage) ClearGauge(name string) {
	fs.MemStorage.ClearGauge(name)
	if fs.pushInterval == 0 {
		if err := fs.saveToFile(); err != nil {
			fs.logger.Error("cant create file", zap.String("file", fs.file), zap.Error(err))
		}
	}
}

func (fs *FileStorage) ClearGauges() {
	fs.MemStorage.ClearGauges()
	if fs.pushInterval == 0 {
		if err := fs.saveToFile(); err != nil {
			fs.logger.Error("cant create file", zap.String("file", fs.file), zap.Error(err))
		}
	}
}

func (fs *FileStorage) SetCounter(name string, value Counter) {
	fs.MemStorage.SetCounter(name, value)
	if fs.pushInterval == 0 {
		if err := fs.saveToFile(); err != nil {
			fs.logger.Error("cant create file", zap.String("file", fs.file), zap.Error(err))
		}
	}
}

func (fs *FileStorage) ClearCounter(name string) {
	fs.MemStorage.ClearCounter(name)
	if fs.pushInterval == 0 {
		if err := fs.saveToFile(); err != nil {
			fs.logger.Error("cant create file", zap.String("file", fs.file), zap.Error(err))
		}
	}
}

func (fs *FileStorage) ClearCounters() {
	fs.MemStorage.ClearCounters()
	if fs.pushInterval == 0 {
		if err := fs.saveToFile(); err != nil {
			fs.logger.Error("cant create file", zap.String("file", fs.file), zap.Error(err))
		}
	}
}
