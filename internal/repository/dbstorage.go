package repository

import (
	"context"
	"database/sql"
)

type DBStorage struct {
    db *sql.DB
}

func NewDBStorage(db *sql.DB) *DBStorage {
    return &DBStorage{db: db}
}

func (s *DBStorage) Ping(ctx context.Context) error {
    return s.db.PingContext(ctx)
}
