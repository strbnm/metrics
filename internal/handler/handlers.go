package handler

import (
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/strbnm/metrics/internal/model"
	"github.com/strbnm/metrics/internal/repository"
)

type Handler struct {
	repo repository.Repository
}

func NewHandler(repo repository.Repository) *Handler {
	return &Handler{repo: repo}
}

func (h *Handler) UpdateHandler(w http.ResponseWriter, r *http.Request) {
	metricType := chi.URLParam(r, "metricType")
	metricName := chi.URLParam(r, "metricName")
	valueStr := chi.URLParam(r, "metricValue")

	if metricName == "" {
		http.Error(w, "Metric name is required", http.StatusNotFound)
		return
	}

	if valueStr == "" {
		http.Error(w, "Metric value is required", http.StatusBadRequest)
		return
	}

	var metric models.Metrics

	switch metricType {
	case models.Gauge:
		value, err := strconv.ParseFloat(valueStr, 64)
		if err != nil {
			http.Error(w, "Invalid gauge value", http.StatusBadRequest)
			return
		}
		metric = models.Metrics{
			ID:    metricName,
			MType: models.Gauge,
			Value: &value,
		}
	case models.Counter:
		delta, err := strconv.ParseInt(valueStr, 10, 64)
		if err != nil {
			http.Error(w, "Invalid counter value", http.StatusBadRequest)
			return
		}
		metric = models.Metrics{
			ID:    metricName,
			MType: models.Counter,
			Delta: &delta,
		}
	default:
		http.Error(w, "Invalid metric type", http.StatusBadRequest)
		return
	}

	err := h.repo.Save(metric)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "OK")

}

func (h *Handler) ListAllMetricsHandler(w http.ResponseWriter, r *http.Request) {
	metrics, err := h.repo.List()
	if err != nil {
		http.Error(w, "Failed to get metrics", http.StatusInternalServerError)
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

	metric, err := h.repo.Get(metricName, metricType)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")

	switch metric.MType {
	case models.Gauge:
		fmt.Fprintf(w, "%.6f", *metric.Value)
	case models.Counter:
		fmt.Fprint(w, *metric.Delta)
	}
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
				valueStr = fmt.Sprintf("%.6f", *metric.Value)
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
