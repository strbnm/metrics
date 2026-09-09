package main

import (
	"errors"
	"net/http"

	config "github.com/strbnm/metrics/internal/config/server"
	"github.com/strbnm/metrics/internal/handler"
	"github.com/strbnm/metrics/internal/logger"
	"github.com/strbnm/metrics/internal/middleware"
	"github.com/strbnm/metrics/internal/repository"
	"github.com/strbnm/metrics/internal/service"

	"github.com/go-chi/chi/v5"
)

func main() {
	cleanup := logger.SetupLogger("info")
	defer cleanup()

	if err := run(); err != nil {
		logger.Log.Errorf("server error %v", err)
		panic(err)
	}
}

func run() error {
	cfg, err := config.ParseFlags()
	if err != nil {
		logger.Log.Errorf("Configuration parsing failed: %v", err)
		return errors.New("configuration parsing failed")
	}
	memStorage := repository.NewMemStorage()
	metricsService := service.NewMetricsService(memStorage)

	h := handler.NewHandler(metricsService)

	r := chi.NewRouter()
	r.Use(middleware.WithLogging)
	r.Route("/", func(r chi.Router) {
		r.Get("/", h.ListAllMetricsHandler)
		r.Get("/value/{metricType}/{metricName}", h.ValueHandler)
		r.Post("/update/{metricType}/{metricName}/{metricValue}", h.UpdateHandler)
	})

	logger.Log.Infof("Server is starting on %s", cfg.RunAddr)

	return http.ListenAndServe(cfg.RunAddr, r)
}
