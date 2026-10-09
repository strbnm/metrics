package service

import (
	"context"
	"log"
	"sync"
	"time"

	models "github.com/strbnm/metrics/internal/model"
)

type Collector struct {
	rateLimit      int
	pollInterval   time.Duration
	reportInterval time.Duration

	lastRuntime []models.Metrics
	lastSystem  []models.Metrics

	sendQueue chan []models.Metrics
	mu        sync.RWMutex
	wg        sync.WaitGroup
}

func NewCollector(pollInterval, reportInterval time.Duration, rateLimit int) *Collector {
	c := &Collector{
		pollInterval:   pollInterval,
		reportInterval: reportInterval,
		rateLimit:      rateLimit,
		sendQueue:      make(chan []models.Metrics, 100), // буфер очереди
	}
	return c
}

// runtimeCollector — отдельная горутина: собирает runtime-метрики и сохраняет локально
func (c *Collector) runtimeCollector(ctx context.Context, pollTicker <-chan time.Time) {
	var pollCount, lastSentPollCount int64

	for {
		select {
		case <-ctx.Done():
			return
		case <-pollTicker:
			pollCount++
			rtMetrics := collectRuntimeMetrics(pollCount, lastSentPollCount)

			c.mu.Lock()
			c.lastRuntime = rtMetrics
			c.mu.Unlock()
		}
	}
}

// systemCollector — отдельная горутина: собирает system-метрики и сохраняет локально
func (c *Collector) systemCollector(ctx context.Context, pollTicker <-chan time.Time) {

	for {
		select {
		case <-ctx.Done():
			return
		case <-pollTicker:
			sysMetrics := collectSystemMetrics()

			c.mu.Lock()
			c.lastSystem = sysMetrics
			c.mu.Unlock()
		}
	}
}

// worker — воркер из пула: забирает пакеты из очереди и вызывает sendFunc
func (c *Collector) worker(ctx context.Context, sendFunc func(ctx context.Context, metrics []models.Metrics) error) {
	defer c.wg.Done()
	for {
		select {
		case metrics, ok := <-c.sendQueue:
			if !ok {
				return
			}
			err := sendFunc(ctx, metrics)
			if err != nil {
				log.Printf("Failed to send metrics batch: %v", err)
			}
		case <-ctx.Done():
			return
		}
	}
}

func (c *Collector) Start(ctx context.Context, sendFunc func(ctx context.Context, metrics []models.Metrics) error) error {
	// Инициализируем пул воркеров: ровно rateLimit одновременных вызовов sendFunc
	for w := 0; w < c.rateLimit; w++ {
		c.wg.Add(1)
		go c.worker(ctx, sendFunc)
	}

	pollTicker := time.NewTicker(c.pollInterval)
	defer pollTicker.Stop()

	reportTicker := time.NewTicker(c.reportInterval)
	defer reportTicker.Stop()

	// Горутина сбора runtime-метрик (без отправки)
	go c.runtimeCollector(ctx, pollTicker.C)

	// Горутина сбора system-метрик (без отправки)
	go c.systemCollector(ctx, pollTicker.C)

	for {
		select {
		case <-ctx.Done():
			close(c.sendQueue)
			c.wg.Wait()
			return nil
		case <-reportTicker.C:
			// Снимаем снапшоты под одним RLock
			c.mu.RLock()
			rtSnapshot := append([]models.Metrics(nil), c.lastRuntime...)
			sysSnapshot := append([]models.Metrics(nil), c.lastSystem...)
			c.mu.RUnlock()

			// Отправляем runtime-пакет отдельным сообщением в очередь
			select {
			case c.sendQueue <- rtSnapshot:
			case <-ctx.Done():
				return nil
			}

			// Отправляем system-пакет отдельным сообщением в очередь
			select {
			case c.sendQueue <- sysSnapshot:
			case <-ctx.Done():
				return nil
			}
		}
	}
}
