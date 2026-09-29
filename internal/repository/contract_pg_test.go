//go:build integration

package repository

import (
	"context"
	"database/sql"
	"os"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/strbnm/metrics/internal/service"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/strbnm/metrics/migrations"
)

func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()

	dsn := os.Getenv("TEST_DATABASE_DSN")
	if dsn == "" {
		dsn = "host=localhost port=5433 user=postgres password=postgres dbname=test_metrics sslmode=disable"
	}

	db, err := sql.Open("pgx", dsn)
	require.NoError(t, err)

	// применяем миграции
	goose.SetBaseFS(migrations.Files)
	require.NoError(t, goose.SetDialect("postgres"))
	require.NoError(t, goose.Up(db, "."))

	return db
}

// cleanupDB — очищает таблицу перед каждым тестом
func cleanupDB(t *testing.T, db *sql.DB) {
	t.Helper()
	_, err := db.Exec("DELETE FROM metrics")
	require.NoError(t, err)
}

func TestPostgres_Contract(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	testRepositoryContract(ctx, t, func() service.Repository {
		cleanupDB(t, db) // чистая БД для каждого под-теста
		testLogger := zap.NewNop().Sugar()

		return NewDBStorage(db, testLogger)
	})
}
