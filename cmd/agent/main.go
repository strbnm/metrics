package main

import (
	"log"
	"time"

	"github.com/strbnm/metrics/internal/agent"
	models "github.com/strbnm/metrics/internal/model"
	"github.com/strbnm/metrics/internal/service"
)

func main() {
	pollInterval := 2 * time.Second
	reportInterval := 10 * time.Second
	serverURL := "http://localhost:8080"

	collector := service.NewCollector(pollInterval, reportInterval)

	sender := agent.NewSender(serverURL)

	log.Printf("Starting metrics agent: poll every %v, report every %v to %s",
		pollInterval, reportInterval, serverURL)

	collector.Start(func(metrics []models.Metrics) error {
		return sender.Send(metrics)
	})
}
