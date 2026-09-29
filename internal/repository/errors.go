package repository

import "errors"

var (
	ErrMetricNotFound            = errors.New("repository: metric not found")
	ErrInvalidMetricType         = errors.New("repository: unknown metric type")
	ErrGaugeMetricIsEmptyValue   = errors.New("repository: gauge metric must have a value")
	ErrCounterMetricIsEmptyValue = errors.New("repository: counter metric must have a value")
)
