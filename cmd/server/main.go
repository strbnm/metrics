package main

import (
	"log"
	"net/http"

	"github.com/strbnm/metrics/internal/handler"
	"github.com/strbnm/metrics/internal/repository"
)

func main() {
	memStorage := repository.NewMemStorage()

	h := handler.NewHandler(memStorage)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /update/{metricType}/{metricName}/{metricValue}/", h.UpdateHandler)

	log.Println("Server is starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
