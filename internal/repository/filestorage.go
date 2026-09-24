package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"time"

	models "github.com/strbnm/metrics/internal/model"
	"go.uber.org/zap"
)

type FileStorage struct {
	*MemStorage
	fileName    string
	isSyncFlush bool
	logger      *zap.SugaredLogger
}

func NewFileStorage(fileName string, isSyncFlush, restore bool, logger *zap.SugaredLogger) *FileStorage {
	fileStorage := &FileStorage{
		MemStorage:  NewMemStorage(),
		fileName:    fileName,
		isSyncFlush: isSyncFlush,
		logger:      logger,
	}
	// Загрузка из файла если restore == true
	if restore {
		initCtx, initCancel := context.WithTimeout(context.Background(), 5*time.Second)

		// Читаем файл в горутине, записываем результат только при успехе
		done := make(chan struct {
			metrics map[string]models.Metrics
			err     error
		}, 1)

		go func() {
			loaded, err := readMetricsFromFile(fileName)
			done <- struct {
				metrics map[string]models.Metrics
				err     error
			}{loaded, err}
		}()

		select {
		case res := <-done:
			if res.err != nil {
				logger.Errorf("failed to load metrics: %v", res.err)
			} else {
				fileStorage.mu.Lock()
				fileStorage.MemStorage.metrics = res.metrics
				fileStorage.mu.Unlock()
				logger.Infow("metrics restored from file", "filename", fileName)
			}
		case <-initCtx.Done():
			logger.Warnf("metrics load timed out, starting with empty storage")
		}

		initCancel()
	}

	return fileStorage
}

func (r *FileStorage) Save(ctx context.Context, metric models.Metrics) error {
	err := r.MemStorage.Save(ctx, metric)
	if err != nil {
		return err
	}
	if r.isSyncFlush {
		err = r.Flush(ctx)
		if err != nil {
			return err
		}
	}
	return nil
}

func readMetricsFromFile(fileName string) (map[string]models.Metrics, error) {
	file, err := os.Open(fileName)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("repository: open file: %w", err)
	}
	defer file.Close()

	var loaded []models.Metrics
	decoder := json.NewDecoder(file)
	if errDecode := decoder.Decode(&loaded); errDecode != nil {
		if errors.Is(errDecode, io.EOF) {
			return nil, nil
		}
		return nil, fmt.Errorf("repository: decode metrics: %w", errDecode)
	}

	metrics := make(map[string]models.Metrics, len(loaded))
	for _, metric := range loaded {
		key := buildKey(metric.ID, metric.MType)
		metrics[key] = metric
	}
	return metrics, nil
}

func (r *FileStorage) Flush(ctx context.Context) error {
	listMetrics, err := r.MemStorage.List(ctx)
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
