package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/go-chi/chi/v5"
	models "github.com/strbnm/metrics/internal/model"
	"github.com/strbnm/metrics/internal/service"
	"go.uber.org/zap"
)

type MetricsService interface {
	UpdateMetric(ctx context.Context, metricType, metricName, valueStr string) error
	ListAllMetrics(ctx context.Context) ([]models.Metrics, error)
	GetMetricValue(ctx context.Context, metricName, metricType string) (string, error)
	UpdateMetricFromModel(ctx context.Context, m models.Metrics) error
	GetMetric(ctx context.Context, m *models.Metrics) error
}

type Handler struct {
	service MetricsService
	logger  *zap.SugaredLogger
}

func NewHandler(service MetricsService, logger *zap.SugaredLogger) *Handler {
	return &Handler{
		service: service,
		logger:  logger,
	}

}

func (h *Handler) UpdateHandler(w http.ResponseWriter, r *http.Request) {
	metricType := chi.URLParam(r, "metricType")
	metricName := chi.URLParam(r, "metricName")
	valueStr := chi.URLParam(r, "metricValue")

	err := h.service.UpdateMetric(r.Context(), metricType, metricName, valueStr)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidMetricValue):
			http.Error(w, "Invalid metric value", http.StatusBadRequest)
		case errors.Is(err, service.ErrEmptyMetricName):
			http.Error(w, "Metric name is required", http.StatusNotFound)
		case errors.Is(err, service.ErrEmptyMetricValue):
			http.Error(w, "Metric value is required", http.StatusBadRequest)
		case errors.Is(err, service.ErrInvalidMetricType):
			http.Error(w, "Invalid metric type", http.StatusBadRequest)
		default:
			fmt.Printf("Error update metric: %s", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "OK")
}

func (h *Handler) ListAllMetricsHandler(w http.ResponseWriter, r *http.Request) {
	metrics, err := h.service.ListAllMetrics(r.Context())
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	textContent := generateMetricsText(metrics)

	_, err = w.Write([]byte(textContent))
	if err != nil {
		http.Error(w, "Failed to send response", http.StatusInternalServerError)
	}
}

func (h *Handler) ValueHandler(w http.ResponseWriter, r *http.Request) {
	metricType := chi.URLParam(r, "metricType")
	metricName := chi.URLParam(r, "metricName")

	if metricType != models.Counter && metricType != models.Gauge {
		http.Error(w, "Invalid metric type", http.StatusBadRequest)
		return
	}

	if metricName == "" {
		http.Error(w, "Metric name is required", http.StatusNotFound)
	}

	metricValue, err := h.service.GetMetricValue(r.Context(), metricName, metricType)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprint(w, metricValue)
}

func generateMetricsText(metrics []models.Metrics) string {
	var sb strings.Builder

	sort.Slice(metrics, func(i, j int) bool {
		return metrics[i].ID < metrics[j].ID
	})

	for _, metric := range metrics {
		var valueStr string
		switch metric.MType {
		case models.Gauge:
			if metric.Value != nil {
				valueStr = fmt.Sprint(*metric.Value)
			} else {
				valueStr = "NaN"
			}
		case models.Counter:
			if metric.Delta != nil {
				valueStr = fmt.Sprintf("%d", *metric.Delta)
			} else {
				valueStr = "NaN"
			}
		default:
			valueStr = "unknown"
		}

		sb.WriteString(fmt.Sprintf("%s - %s\n", metric.ID, valueStr))
	}

	return sb.String()
}

func (h *Handler) UpdateJSONHandler(w http.ResponseWriter, r *http.Request) {
	var metric models.Metrics

	if err := json.NewDecoder(r.Body).Decode(&metric); err != nil {
		h.logger.Debugw("Error decoding JSON", "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err := h.service.UpdateMetricFromModel(r.Context(), metric)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrEmptyMetricValue):
			http.Error(w, "Metric value is required", http.StatusBadRequest)
		default:
			fmt.Printf("Error update metric: %s", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "OK")
}

func (h *Handler) ValueJSONHandler(w http.ResponseWriter, r *http.Request) {
	var m models.Metrics

	if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
		h.logger.Debugw("Error decoding JSON", "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err := h.service.GetMetric(r.Context(), &m)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	err = json.NewEncoder(w).Encode(m)
	if err != nil {
		return
	}
}
