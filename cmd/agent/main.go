package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

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

	collector := service.NewCollector(cfg.CollectorConfig.PollInterval, cfg.CollectorConfig.ReportInterval, cfg.CollectorConfig.RateLimit)

	sender, err := agent.NewSender(serverURL, cfg.ServerConfig.SignKey)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Starting metrics agent: poll every %v, report every %v to %s",
		cfg.CollectorConfig.PollInterval, cfg.CollectorConfig.ReportInterval, serverURL)

	ctx, stop := signal.NotifyContext(context.Background(),
		syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	collector.Start(ctx, func(ctx context.Context, metrics []models.Metrics) error {
		return sender.Send(ctx, metrics)
	})
}
