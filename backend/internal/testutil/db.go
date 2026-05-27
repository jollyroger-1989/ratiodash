// Package testutil provides shared helpers for backend tests.
package testutil

import (
	"database/sql"
	"path/filepath"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/jose/ratiodash/migrations"
)

// NewDB creates a fresh SQLite database in a temporary directory, runs all
// embedded Goose migrations, and returns a *gorm.DB ready for use in tests.
// Each call produces a fully isolated database; the file is removed automatically
// when the test (or sub-test) finishes.
func NewDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := filepath.Join(t.TempDir(), "test.db")

	sqlDB, err := sql.Open("sqlite3", dsn)
	require.NoError(t, err, "open sqlite3 db")
	t.Cleanup(func() { _ = sqlDB.Close() })

	goose.SetBaseFS(migrations.FS)
	require.NoError(t, goose.SetDialect("sqlite3"), "set goose dialect")
	require.NoError(t, goose.Up(sqlDB, "."), "run migrations")

	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err, "open gorm db")

	return db
}
