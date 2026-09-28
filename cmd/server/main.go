package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/strbnm/metrics/internal/config/db"
	config "github.com/strbnm/metrics/internal/config/server"
	"github.com/strbnm/metrics/internal/handler"
	"github.com/strbnm/metrics/internal/logger"
	"github.com/strbnm/metrics/internal/middleware"
	"github.com/strbnm/metrics/internal/repository"
	"github.com/strbnm/metrics/internal/repository/postgres"
	"github.com/strbnm/metrics/internal/service"
	"go.uber.org/zap"

	"github.com/go-chi/chi/v5"

	middlewares "github.com/go-chi/chi/v5/middleware"
)

const (
	shutdownTimeout   = 5 * time.Second
	flushTimeout      = 5 * time.Second
	readHeaderTimeout = 5 * time.Second
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

func run() error {
	cfg, err := config.ParseFlags()
	if err != nil {
		return fmt.Errorf("main: configuration parsing failed %w", err)
	}

	appLogger, cleanup, err := logger.SetupLogger("info")
	if err != nil {
		return fmt.Errorf("main: logger initialization failed %w", err)
	}
	defer cleanup()

	ctx, stop := signal.NotifyContext(context.Background(),
		syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	storage, pinger, fileStorage, cleanupStorage, err := newStorage(ctx, cfg, appLogger)
	if err != nil {
		return fmt.Errorf("main: %w", err)
	}
	defer cleanupStorage()

	metricsService := service.NewMetricsService(storage, appLogger)

	h := handler.NewHandler(metricsService, pinger, appLogger)

	srv := &http.Server{
		Addr:              cfg.RunAddr,
		Handler:           newRouter(h, appLogger),
		ReadHeaderTimeout: readHeaderTimeout,
	}

	appLogger.Infof("Server is starting on %s", cfg.RunAddr)

	// Запуск сервера в горутине, чтобы main мог ждать сигнал
	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.ListenAndServe()
	}()

	// Ждём сигнал остановки или ошибку сервера
	select {
	case <-ctx.Done():
		appLogger.Info("shutting down gracefully...")
	case errServer := <-errCh:
		if errServer != nil && !errors.Is(errServer, http.ErrServerClosed) {
			return errServer
		}
	}

	// Останавливаем сервер с таймаутом
	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		appLogger.Errorf("server shutdown error: %v", err)
	}

	// Финальное сохранение метрик
	flushFileStorage(fileStorage, appLogger)

	return nil
}

// newStorage создаёт хранилище по приоритету: БД → файл → память.
// Возвращает также функцию освобождения ресурсов и файловое хранилище,
// если оно было создано (для финального сохранения).
func newStorage(
	ctx context.Context,
	cfg *config.Config,
	appLogger *zap.SugaredLogger,
) (storage service.Repository, pinger handler.Pinger, fileStorage *repository.FileStorage, cleanup func(), err error) {
	cleanup = func() {}

	switch {
	case cfg.DatabaseConfig.DatabaseDSN != "":
		// приоритет 1 — база данных
		dbCfg := db.DefaultConfig(cfg.DatabaseConfig.DatabaseDSN)
		dbConn, err := postgres.NewPostgresConnection(dbCfg)
		if err != nil {
			return nil, nil, nil, cleanup, fmt.Errorf("db connection failed: %w", err)
		}

		if err := postgres.RunMigrations(dbConn); err != nil {
			dbConn.Close()
			return nil, nil, nil, cleanup, fmt.Errorf("migrations failed: %w", err)
		}
		appLogger.Info("database connection established")

		dbStorage := repository.NewDBStorage(dbConn, appLogger)
		return dbStorage, dbStorage, nil, func() { dbConn.Close() }, nil

	case cfg.Store.FileStoragePath != "":
		// приоритет 2 — файловое хранилище
		isSyncFlush := cfg.Store.StoreInterval == 0
		fileStorage = repository.NewFileStorage(
			cfg.Store.FileStoragePath, isSyncFlush, cfg.Store.Restore, appLogger,
		)

		if !isSyncFlush {
			go runPeriodicSave(ctx, fileStorage, cfg.Store.StoreInterval, appLogger)
		}
		appLogger.Info("file storage initialized", "path", cfg.Store.FileStoragePath)
		return fileStorage, nil, fileStorage, cleanup, nil

	default:
		// приоритет 3 — in-memory
		storage = repository.NewMemStorage()
		appLogger.Info("in-memory storage initialized")
		return storage, nil, nil, cleanup, nil
	}
}

// newRouter собирает цепочку middleware и маршруты API.
func newRouter(h *handler.Handler, appLogger *zap.SugaredLogger) *chi.Mux {
	r := chi.NewRouter()
	r.Use(middlewares.StripSlashes)
	r.Use(middleware.GzipMiddleware)
	r.Use(middleware.WithLogging(appLogger))

	r.Route("/", func(r chi.Router) {
		r.Get("/", h.ListAllMetricsHandler)
		r.Get("/value/{metricType}/{metricName}", h.ValueHandler)
		r.Get("/ping", h.Ping)
		r.Post("/update/{metricType}/{metricName}/{metricValue}", h.UpdateHandler)
		r.Post("/update", h.UpdateJSONHandler)
		r.Post("/updates", h.UpdateBatchJSONHandler)
		r.Post("/value", h.ValueJSONHandler)
	})

	return r
}

// flushFileStorage выполняет финальное сохранение метрик в файл, если оно настроено.
func flushFileStorage(fileStorage *repository.FileStorage, appLogger *zap.SugaredLogger) {
	if fileStorage == nil {
		return
	}
	flushCtx, flushCancel := context.WithTimeout(context.Background(), flushTimeout)
	defer flushCancel()
	if errFlush := fileStorage.Flush(flushCtx); errFlush != nil {
		appLogger.Errorf("final flush failed: %v", errFlush)
	} else {
		appLogger.Info("final flush completed")
	}
}

func runPeriodicSave(ctx context.Context, r *repository.FileStorage, interval time.Duration, appLogger *zap.SugaredLogger) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := r.Flush(ctx); err != nil {
				appLogger.Errorf("periodic save failed: %v", err)
			} else {
				appLogger.Infow("periodic save completed")
			}
		case <-ctx.Done():
			return
		}
	}
}
