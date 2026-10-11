package service

import (
	"context"
	"log"
	"sync"
	"sync/atomic"
	"time"

	models "github.com/strbnm/metrics/internal/model"
)

type Collector struct {
	rateLimit      int
	pollInterval   time.Duration
	reportInterval time.Duration

	lastRuntime []models.Metrics
	lastSystem  []models.Metrics

	// lastSentPollCount — абсолютный pollCount последнего подтверждённого
	// (успешно отправленного) runtime-батча; обновляют воркеры через CAS-max.
	lastSentPollCount atomic.Int64
	// runtimeInFlight — гейт «не больше одного runtime-батча в полёте»:
	// пока он не подтверждён или окончательно не провален, следующая дельта
	// не вырезается, чтобы интервалы батчей не пересекались.
	runtimeInFlight atomic.Bool

	sendQueue chan []models.Metrics
	mu        sync.RWMutex
	wg        sync.WaitGroup
}

func NewCollector(pollInterval, reportInterval time.Duration, rateLimit int) *Collector {
	c := &Collector{
		pollInterval:   pollInterval,
		reportInterval: reportInterval,
		rateLimit:      rateLimit,
		sendQueue:      make(chan []models.Metrics, rateLimit), // буфер очереди
	}
	return c
}

// runtimeCollector — отдельная горутина: собирает runtime-метрики и сохраняет локально
func (c *Collector) runtimeCollector(ctx context.Context, pollTicker <-chan time.Time) {
	var pollCount int64

	for {
		select {
		case <-ctx.Done():
			return
		case <-pollTicker:
			pollCount++
			// Снапшот несёт кумулятивный счётчик (lastSent = 0):
			// непересекающуюся дельту вырезает Start при постановке в очередь
			rtMetrics := collectRuntimeMetrics(pollCount, 0)

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
			p, isRuntime := cutPollCountDelta(metrics, c.lastSentPollCount.Load())

			err := sendFunc(ctx, metrics)
			if err != nil {
				log.Printf("Failed to send metrics batch: %v", err)
			}

			if isRuntime {
				if err == nil {
					confirmPollCount(&c.lastSentPollCount, p)
				}
				// Любой финальный исход освобождает гейт: при провале счётчик
				// не подтверждён, и следующий отчёт включит интервал заново
				c.runtimeInFlight.CompareAndSwap(true, false)
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

	runtimeTicker := time.NewTicker(c.pollInterval)
	defer runtimeTicker.Stop()

	systemTicker := time.NewTicker(c.pollInterval)
	defer systemTicker.Stop()

	reportTicker := time.NewTicker(c.reportInterval)
	defer reportTicker.Stop()

	// Горутина сбора runtime-метрик (без отправки)
	go c.runtimeCollector(ctx, runtimeTicker.C)

	// Горутина сбора system-метрик (без отправки)
	go c.systemCollector(ctx, systemTicker.C)

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

			// Runtime-пакет уходит не чаще одного «в полёте»: пока предыдущий
			// не подтвердился или окончательно не провалился, отчёт
			// пропускается, а опросы копятся в кумулятивном счётчике
			if _, ok := pollCountOf(rtSnapshot); ok && c.runtimeInFlight.CompareAndSwap(false, true) {
				select {
				case c.sendQueue <- rtSnapshot:
				case <-ctx.Done():
					c.runtimeInFlight.CompareAndSwap(true, false)
					return nil
				}
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

// pollCountOf возвращает кумулятивное значение PollCount из батча.
// Признак runtime-батча — наличие этой метрики.
func pollCountOf(metrics []models.Metrics) (int64, bool) {
	for _, m := range metrics {
		if m.ID == "PollCount" && m.Delta != nil {
			return *m.Delta, true
		}
	}
	return 0, false
}

// cutPollCountDelta делает за один проход то же, что pollCountOf и
// перезапись дельты: находит в runtime-батче PollCount, вырезает из
// кумулятивного значения интервал поверх lastSent и возвращает кумулятивное
// значение для последующего подтверждения. ok = false означает, что батч
// не runtime (метрики PollCount в нём нет) и он остался без изменений.
// Гейт гарантирует, что пока батч в полёте, lastSent не меняется.
func cutPollCountDelta(metrics []models.Metrics, lastSent int64) (int64, bool) {
	for i := range metrics {
		if metrics[i].ID == "PollCount" && metrics[i].Delta != nil {
			cumulative := *metrics[i].Delta
			metrics[i].Delta = new(cumulative - lastSent)
			return cumulative, true
		}
	}
	return 0, false
}

// confirmPollCount продвигает lastSentPollCount до sent (CAS-max:
// подтверждения могут приходить не по порядку, откат недопустим)
func confirmPollCount(field *atomic.Int64, sent int64) {
	for {
		cur := field.Load()
		if sent <= cur || field.CompareAndSwap(cur, sent) {
			return
		}
	}
}
