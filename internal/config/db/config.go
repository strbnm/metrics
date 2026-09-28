package db

import (
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
		ConnMaxIdleTime:    1 * time.Minute,
	}
}
