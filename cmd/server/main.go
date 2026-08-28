package main

import (
	"log"
	"net/http"

	"github.com/strbnm/metrics/internal/handler"
	"github.com/strbnm/metrics/internal/repository"
	"github.com/strbnm/metrics/internal/service"

	config "github.com/strbnm/metrics/internal/config/server"

	"github.com/go-chi/chi/v5"
)

func main() {
	cfg, err := config.ParseFlags()
	if err != nil {
		log.Fatalf("Configuration parsing failed: %v", err)
	}
	memStorage := repository.NewMemStorage()
	metricsService := service.NewMetricsService(memStorage)

	h := handler.NewHandler(metricsService)

	r := chi.NewRouter()
	r.Route("/", func(r chi.Router) {
		r.Get("/", h.ListAllMetricsHandler)
		r.Get("/value/{metricType}/{metricName}", h.ValueHandler)
		r.Post("/update/{metricType}/{metricName}/{metricValue}", h.UpdateHandler)
	})

	log.Printf("Server is starting on %s", cfg.RunAddr)
	log.Fatal(http.ListenAndServe(cfg.RunAddr, r))
}
