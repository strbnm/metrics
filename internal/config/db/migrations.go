package db

import (
	"database/sql"

	"github.com/pressly/goose/v3"
	"github.com/strbnm/metrics/migrations"
)

func RunMigrations(db *sql.DB) error {
	goose.SetBaseFS(migrations.Files)
	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}
	if err := goose.Up(db, "."); err != nil {
		return err
	}
	return nil
}
