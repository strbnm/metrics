package main

import (
	"log"

	"github.com/strbnm/metrics/internal/agent"
	models "github.com/strbnm/metrics/internal/model"
	"github.com/strbnm/metrics/internal/service"
)

func main() {
	parseFlags()
	serverURL := "http://" + flagRunAddr

	collector := service.NewCollector(flagPollInterval, flagReportInterval)

	sender, err := agent.NewSender(serverURL)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Starting metrics agent: poll every %v, report every %v to %s",
		flagPollInterval, flagReportInterval, serverURL)

	collector.Start(func(metrics []models.Metrics) error {
		return sender.Send(metrics)
	})
}
