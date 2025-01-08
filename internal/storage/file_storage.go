package storage

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"
)

type Metric struct {
	Delta *Counter `json:"delta,omitempty"`
	Value *Gauge   `json:"value,omitempty"`
	ID    string   `json:"id"`
	MType string   `json:"type"`
}

type FileStorage struct {
	MemStorage
	file         string
	pushInterval int
}

func NewFileStorage(file string, pushInterval int, restore bool) (*FileStorage, error) {
	storage := &FileStorage{
		MemStorage:   *NewMemStorage(),
		file:         file,
		pushInterval: pushInterval,
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

	data := []Metric{}
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
				if err := storage.SaveToFile(); err != nil {
					fmt.Printf("cant save: %v\n", err)
				}
			}
		}()
	}

	return storage, nil
}

func (fs *FileStorage) SaveToFile() error {
	f, err := os.Create(fs.file)
	if err != nil {
		return fmt.Errorf("cant create file: %w", err)
	}
	defer func() {
		if closeErr := f.Close(); closeErr != nil {
			err = fmt.Errorf("cant close file: %w", closeErr)
		}
	}()

	data := []Metric{}

	for id, value := range fs.GetAllGauges() {
		metric := Metric{
			ID:    id,
			MType: "gauge",
			Value: &value,
		}
		data = append(data, metric)
	}

	for id, delta := range fs.GetAllCounters() {
		metric := Metric{
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
		if err := fs.SaveToFile(); err != nil {
			log.Printf("Cant save metric: %v", err)
		}
	}
}

func (fs *FileStorage) ClearGauge(name string) {
	fs.MemStorage.ClearGauge(name)
	if fs.pushInterval == 0 {
		if err := fs.SaveToFile(); err != nil {
			log.Printf("Cant save metric: %v", err)
		}
	}
}

func (fs *FileStorage) ClearGauges() {
	fs.MemStorage.ClearGauges()
	if fs.pushInterval == 0 {
		if err := fs.SaveToFile(); err != nil {
			log.Printf("Cant save metric: %v", err)
		}
	}
}

func (fs *FileStorage) SetCounter(name string, value Counter) {
	fs.MemStorage.SetCounter(name, value)
	if fs.pushInterval == 0 {
		if err := fs.SaveToFile(); err != nil {
			log.Printf("Cant save metric: %v", err)
		}
	}
}

func (fs *FileStorage) ClearCounter(name string) {
	fs.MemStorage.ClearCounter(name)
	if fs.pushInterval == 0 {
		if err := fs.SaveToFile(); err != nil {
			log.Printf("Cant save metric: %v", err)
		}
	}
}

func (fs *FileStorage) ClearCounters() {
	fs.MemStorage.ClearCounters()
	if fs.pushInterval == 0 {
		if err := fs.SaveToFile(); err != nil {
			log.Printf("Cant save metric: %v", err)
		}
	}
}
