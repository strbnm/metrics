package service

import "errors"

var (
	ErrInvalidMetricValue = errors.New("service: invalid metric value")
	ErrInvalidMetricType  = errors.New("service: invalid metric type")
	ErrEmptyMetricValue   = errors.New("service: metric value is empty")
	ErrEmptyMetricName    = errors.New("service: metric name is empty")
)
