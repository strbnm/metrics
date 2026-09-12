package repository

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"

	"github.com/strbnm/metrics/internal/logger"
	models "github.com/strbnm/metrics/internal/model"
)

type FileStorage struct {
	*MemStorage
	fileName    string
	isSyncFlush bool
}

func NewFileStorage(fileName string, isSyncFlush, restore bool) *FileStorage {
	fileStorage := &FileStorage{
		MemStorage:  NewMemStorage(),
		fileName:    fileName,
		isSyncFlush: isSyncFlush,
	}
	// Загрузка из файла если restore == true
	if restore {
		if err := fileStorage.Load(); err != nil {
			logger.Log.Errorf("failed to load metrics: %v", err)
		} else {
			logger.Log.Infow("metrics restored from file", "filename", fileName)
		}
	}
	return fileStorage
}

func (r *FileStorage) Save(metric models.Metrics) error {
	err := r.MemStorage.Save(metric)
	if err != nil {
		return err
	}
	if r.isSyncFlush {
		err = r.Flush()
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *FileStorage) Load() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	file, err := os.Open(r.fileName)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil // файла нет — это нормально при первом запуске
		}
		return fmt.Errorf("repository: open file: %w", err)
	}
	defer file.Close()

	var loaded []models.Metrics
	decoder := json.NewDecoder(file)
	if errDecode := decoder.Decode(&loaded); errDecode != nil {
		if errors.Is(err, io.EOF) {
			return nil // пустой файл — ничего не загружаем
		}
		return fmt.Errorf("repository: decode metrics: %w", err)
	}

	metrics := make(map[string]models.Metrics, len(loaded))
	for _, metric := range loaded {
		key := buildKey(metric.ID, metric.MType)
		metrics[key] = metric
	}
	r.MemStorage.metrics = metrics
	logger.Log.Debugf("loaded %d metrics", len(loaded))
	return nil
}

func (r *FileStorage) Flush() error {
	listMetrics, err := r.MemStorage.List()
	if err != nil {
		return err
	}
	sort.Slice(listMetrics, func(i, j int) bool {
		return listMetrics[i].ID < listMetrics[j].ID
	})
	data, err := json.MarshalIndent(listMetrics, "", "  ")
	if err != nil {
		return fmt.Errorf("repository: error marshal metrics: %w", err)
	}

	// Пишем во временный файл, затем переименовываем — атомарно
	tmpPath := r.fileName + ".tmp"
	if errWrite := os.WriteFile(tmpPath, data, 0644); errWrite != nil {
		return fmt.Errorf("repository: write temp file: %w", errWrite)
	}
	if errRename := os.Rename(tmpPath, r.fileName); errRename != nil {
		return fmt.Errorf("repository: rename temp file: %w", errRename)
	}

	return nil
}
