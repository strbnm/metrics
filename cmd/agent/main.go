package main

import (
	"context"
	"log"
	"os"
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

	collector := service.NewCollector(cfg.CollectorConfig.PollInterval, cfg.CollectorConfig.ReportInterval)

	sender, err := agent.NewSender(serverURL)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Starting metrics agent: poll every %v, report every %v to %s",
		cfg.CollectorConfig.PollInterval, cfg.CollectorConfig.ReportInterval, serverURL)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// graceful shutdown: отменяем контекст по сигналу
	go func() {
		stop := make(chan os.Signal, 1)
		signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
		<-stop
		cancel()
	}()

	collector.Start(ctx, func(ctx context.Context, metrics []models.Metrics) error {
		return sender.Send(ctx, metrics)
	})
}
