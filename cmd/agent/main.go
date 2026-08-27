package main

import (
	"log"

	"github.com/strbnm/metrics/internal/agent"
	config "github.com/strbnm/metrics/internal/config/agent"
	models "github.com/strbnm/metrics/internal/model"
	"github.com/strbnm/metrics/internal/service"
)

func main() {
	cfg, err := config.ParseFlags()
	if err != nil {
		log.Fatal(err)
	}
	serverURL := cfg.ServerConfig.ServerURL()

	collector := service.NewCollector(cfg.CollectorConfig.PollInterval, cfg.CollectorConfig.ReportInterval)

	sender, err := agent.NewSender(serverURL)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Starting metrics agent: poll every %v, report every %v to %s",
		cfg.CollectorConfig.PollInterval, cfg.CollectorConfig.ReportInterval, serverURL)

	collector.Start(func(metrics []models.Metrics) error {
		return sender.Send(metrics)
	})
}
