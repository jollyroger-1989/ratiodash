package database

import (
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
	"github.com/pressly/goose/v3"
	"github.com/sirupsen/logrus"
	"go.uber.org/fx"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/jose/ratiodash/migrations"
	"github.com/jose/ratiodash/pkg/config"
)

type gooseLogrusLogger struct{}

func (gooseLogrusLogger) Printf(format string, v ...any) {
	logrus.WithField("component", "goose").Infof(format, v...)
}

func (gooseLogrusLogger) Fatalf(format string, v ...any) {
	logrus.WithField("component", "goose").Fatalf(format, v...)
}

// New opens the SQLite database, runs any pending migrations, then returns a
// configured *gorm.DB for use throughout the application.
func New(cfg *config.Config) (*gorm.DB, error) {
	if err := runMigrations(cfg.DatabaseURL); err != nil {
		return nil, fmt.Errorf("running migrations: %w", err)
	}

	db, err := gorm.Open(sqlite.Open(cfg.DatabaseURL), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	return db, nil
}

// runMigrations applies all pending goose migrations using the embedded SQL files.
func runMigrations(dsn string) error {
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return fmt.Errorf("opening sql db: %w", err)
	}
	defer db.Close()

	goose.SetBaseFS(migrations.FS)
	goose.SetLogger(gooseLogrusLogger{})

	if err := goose.SetDialect("sqlite3"); err != nil {
		return fmt.Errorf("setting dialect: %w", err)
	}

	if err := goose.Up(db, "."); err != nil {
		return fmt.Errorf("applying migrations: %w", err)
	}

	return nil
}

var Module = fx.Options(
	fx.Provide(New),
)
