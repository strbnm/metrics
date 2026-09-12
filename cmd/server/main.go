package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	config "github.com/strbnm/metrics/internal/config/server"
	"github.com/strbnm/metrics/internal/handler"
	"github.com/strbnm/metrics/internal/logger"
	"github.com/strbnm/metrics/internal/middleware"
	"github.com/strbnm/metrics/internal/repository"
	"github.com/strbnm/metrics/internal/service"

	"github.com/go-chi/chi/v5"

	middlewares "github.com/go-chi/chi/v5/middleware"
)

func main() {
	cleanup := logger.SetupLogger("info")
	defer cleanup()

	if err := run(); err != nil {
		logger.Log.Errorf("server error %v", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.ParseFlags()
	if err != nil {
		return fmt.Errorf("configuration parsing failed %w", err)
	}

	isSyncFlush := cfg.Store.StoreInterval == 0
	fileStorage := repository.NewFileStorage(cfg.Store.FileStoragePath, isSyncFlush, cfg.Store.Restore)
	metricsService := service.NewMetricsService(fileStorage)

	ctx, stop := signal.NotifyContext(context.Background(),
		syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Запускаем периодическое сохранение в горутине
	if !isSyncFlush {
		go runPeriodicSave(ctx, fileStorage, cfg.Store.StoreInterval)
	}

	h := handler.NewHandler(metricsService)

	r := chi.NewRouter()
	r.Use(middlewares.StripSlashes)
	r.Use(middleware.GzipMiddleware)
	r.Use(middleware.WithLogging)

	r.Route("/", func(r chi.Router) {
		r.Get("/", h.ListAllMetricsHandler)
		r.Get("/value/{metricType}/{metricName}", h.ValueHandler)
		r.Post("/update/{metricType}/{metricName}/{metricValue}", h.UpdateHandler)
		r.Post("/update", h.UpdateJSONHandler)
		r.Post("/value", h.ValueJSONHandler)
	})

	srv := &http.Server{
		Addr:    cfg.RunAddr,
		Handler: r,
	}

	logger.Log.Infof("Server is starting on %s", cfg.RunAddr)

	// Запуск сервера в горутине, чтобы main мог ждать сигнал
	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.ListenAndServe()
	}()

	// Ждём сигнал остановки или ошибку сервера
	select {
	case <-ctx.Done():
		logger.Log.Info("shutting down gracefully...")
	case errServer := <-errCh:
		if errServer != nil && !errors.Is(errServer, http.ErrServerClosed) {
			return errServer
		}
	}

	// Останавливаем сервер с таймаутом
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Log.Errorf("server shutdown error: %v", err)
	}

	// Финальное сохранение метрик
	if errFlush := fileStorage.Flush(); errFlush != nil {
		logger.Log.Errorf("final flush failed: %v", errFlush)
	} else {
		logger.Log.Info("final flush completed")
	}

	return nil
}

func runPeriodicSave(ctx context.Context, r *repository.FileStorage, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := r.Flush(); err != nil {
				logger.Log.Errorf("periodic save failed: %v", err)
			} else {
				logger.Log.Infow("periodic save completed")
			}
		case <-ctx.Done():
			return
		}
	}
}
