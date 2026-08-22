package handler

import (
	"fmt"
	"net/http"
	"strconv"

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
	metricType := r.PathValue("metricType")
	metricName := r.PathValue("metricName")
	valueStr := r.PathValue("metricValue")

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

	w.Header().Set("Content-Type", "ttext/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "OK")

}
