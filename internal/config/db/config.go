package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type DatabaseConfig struct {
    DatabaseDSN        string
    MaxOpenConnections int
    MaxIdleConnections int
    ConnMaxLifetime    time.Duration
    ConnMaxIdleTime    time.Duration
}

func DefaultConfig(dsn string) *DatabaseConfig {
    return &DatabaseConfig{
        DatabaseDSN:        dsn,
        MaxOpenConnections: 10,
        MaxIdleConnections: 5,
        ConnMaxLifetime:    5 * time.Minute,
        ConnMaxIdleTime:   1 * time.Minute,
    }
}

func NewPostgresConnection(cfg *DatabaseConfig) (*sql.DB, error) {
    db, err := sql.Open("pgx", cfg.DatabaseDSN)
    if err != nil {
        return nil, fmt.Errorf("db: open failed: %w", err)
    }

    db.SetMaxOpenConns(cfg.MaxOpenConnections)
    db.SetMaxIdleConns(cfg.MaxIdleConnections)
    db.SetConnMaxLifetime(cfg.ConnMaxLifetime)
    db.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)

    pingCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
    defer cancel()
    if err := db.PingContext(pingCtx); err != nil {
        db.Close()
        return nil, fmt.Errorf("db: ping failed: %w", err)
    }

    return db, nil
}
