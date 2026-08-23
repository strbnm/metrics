package main

import (
	"log"
	"net/http"

	"github.com/strbnm/metrics/internal/handler"
	"github.com/strbnm/metrics/internal/repository"

	"github.com/go-chi/chi/v5"
)

func main() {
	memStorage := repository.NewMemStorage()

	h := handler.NewHandler(memStorage)

	r := chi.NewRouter()
	r.Route("/", func(r chi.Router) {
		r.Get("/", h.ListAllMetricsHandler)
		r.Get("/value/{metricType}/{metricName}", h.ValueHandler)
		r.Post("/update/{metricType}/{metricName}/{metricValue}", h.UpdateHandler)
	})

	log.Println("Server is starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}
