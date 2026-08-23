package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	models "github.com/strbnm/metrics/internal/model"
)

func TestNewCollector(t *testing.T) {
	pollInterval := 1 * time.Second
	reportInterval := 5 * time.Second

	collector := NewCollector(pollInterval, reportInterval)

	require.NotNil(t, collector, "NewCollector returned nil")
	assert.Equal(t, pollInterval, collector.pollInterval, "pollInterval mismatch")
	assert.Equal(t, reportInterval, collector.reportInterval, "reportInterval mismatch")
	assert.Equal(t, int64(0), collector.pollCount, "initial pollCount should be 0")
	assert.False(t, collector.lastSendTime.IsZero(), "lastSendTime should be set to current time")
}

func TestUint64ToFloat64(t *testing.T) {
	testCases := []struct {
		input    uint64
		expected float64
	}{
		{0, 0},
		{1, 1},
		{42, 42},
		{18446744073709551615, 18446744073709551615},
	}

	for _, tc := range testCases {
		result := uint64ToFloat64(tc.input)
		require.NotNil(t, result, "uint64ToFloat64(%d) returned nil", tc.input)
		assert.InDelta(t, tc.expected, *result, 0.0001, "uint64ToFloat64(%d) = %f, want %f", tc.input, *result, tc.expected)
	}
}

func TestUint32ToFloat64(t *testing.T) {
	testCases := []struct {
		input    uint32
		expected float64
	}{
		{0, 0},
		{1, 1},
		{42, 42},
		{4294967295, 4294967295},
	}

	for _, tc := range testCases {
		result := uint32ToFloat64(tc.input)
		require.NotNil(t, result, "uint32ToFloat64(%d) returned nil", tc.input)
		assert.InDelta(t, tc.expected, *result, 0.0001, "uint32ToFloat64(%d) = %f, want %f", tc.input, *result, tc.expected)
	}
}

func TestCollector_collectOnce(t *testing.T) {
	collector := NewCollector(1*time.Second, 5*time.Second)

	// Первый сбор
	collector.collectOnce()
	firstMetrics := collector.GetLastMetrics()

	require.NotEmpty(t, firstMetrics, "collectOnce produced no metrics")
	assert.Equal(t, int64(1), collector.pollCount, "pollCount should be 1 after first collectOnce")

	var pollCountMetric *models.Metrics
	for _, m := range firstMetrics {
		if m.ID == "PollCount" {
			pollCountMetric = &m
			break
		}
	}

	require.NotNil(t, pollCountMetric, "PollCount metric not found in collected metrics")
	require.NotNil(t, pollCountMetric.Delta, "PollCount Delta should not be nil")
	assert.Equal(t, int64(1), *pollCountMetric.Delta, "PollCount metric should have value 1")

	// Второй сбор
	collector.collectOnce()
	secondMetrics := collector.GetLastMetrics()

	assert.Equal(t, int64(2), collector.pollCount, "pollCount should be 2 after second collectOnce")

	for _, m := range secondMetrics {
		if m.ID == "PollCount" {
			require.NotNil(t, m.Delta, "PollCount Delta should not be nil")
			assert.Equal(t, int64(2), *m.Delta, "PollCount metric should have value 2")
		}
	}

	var randomValueMetric *models.Metrics
	for _, m := range secondMetrics {
		if m.ID == "RandomValue" {
			randomValueMetric = &m
			break
		}
	}

	require.NotNil(t, randomValueMetric, "RandomValue metric not found")
	require.NotNil(t, randomValueMetric.Value, "RandomValue Value should not be nil")
	value := *randomValueMetric.Value
	assert.GreaterOrEqual(t, value, 0.0, "RandomValue should be >= 0")
	assert.Less(t, value, 1.0, "RandomValue should be < 1")
}

func TestCollector_GetLastMetrics(t *testing.T) {
	collector := NewCollector(1*time.Second, 5*time.Second)

	// Изначально метрик нет
	metrics := collector.GetLastMetrics()
	assert.Nil(t, metrics, "Expected no metrics before collectOnce")

	// После сбора должны быть метрики
	collector.collectOnce()
	metrics = collector.GetLastMetrics()
	require.NotEmpty(t, metrics, "GetLastMetrics returned empty slice after collectOnce")

	// Проверяем наличие ключевых метрик
	requiredMetrics := map[string]bool{
		"Alloc":       false,
		"HeapAlloc":   false,
		"PollCount":   false,
		"RandomValue": false,
	}

	for _, m := range metrics {
		if _, exists := requiredMetrics[m.ID]; exists {
			requiredMetrics[m.ID] = true
		}
	}

	for metric, found := range requiredMetrics {
		assert.True(t, found, "Required metric %s not found in collected metrics", metric)
	}
}

func TestCollector_Start_SingleIteration(t *testing.T) {
	pollInterval := 10 * time.Millisecond
	reportInterval := 20 * time.Millisecond

	collector := NewCollector(pollInterval, reportInterval)

	sentMetrics := make([][]models.Metrics, 0)
	sendFunc := func(metrics []models.Metrics) error {
		sentMetrics = append(sentMetrics, metrics)
		return nil
	}

	// Имитируем одну итерацию цикла Start
	collector.collectOnce()

	// Пропускаем время, чтобы условие отправки выполнилось
	collector.lastSendTime = collector.lastSendTime.Add(-reportInterval - 1*time.Millisecond)

	collector.collectOnce()

	if time.Since(collector.lastSendTime) >= collector.reportInterval {
		metrics := collector.GetLastMetrics()
		err := sendFunc(metrics)
		require.NoError(t, err, "sendFunc returned error")
		collector.lastSendTime = time.Now()
	}

	// Проверяем, что метрики были отправлены
	require.Len(t, sentMetrics, 1, "Expected exactly 1 metric send")
	assert.NotEmpty(t, sentMetrics[0], "Sent metrics should not be empty")
}
