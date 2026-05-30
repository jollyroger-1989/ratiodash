package main

import (
	"os"

	"github.com/sirupsen/logrus"

	"github.com/jose/ratiodash/internal/handler"
	"github.com/jose/ratiodash/internal/notifier"
	"github.com/jose/ratiodash/internal/repository"
	"github.com/jose/ratiodash/internal/scheduler"
	"github.com/jose/ratiodash/internal/scraper"
	"github.com/jose/ratiodash/internal/server"
	"github.com/jose/ratiodash/internal/service"
	"github.com/jose/ratiodash/pkg/config"
	"github.com/jose/ratiodash/pkg/database"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
)

func configureLogger() {
	logrus.SetOutput(os.Stdout)
	logrus.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})

	level := logrus.InfoLevel
	if raw := os.Getenv("LOG_LEVEL"); raw != "" {
		if parsed, err := logrus.ParseLevel(raw); err == nil {
			level = parsed
		}
	}
	logrus.SetLevel(level)
}

func main() {
	configureLogger()

	fx.New(
		fx.WithLogger(func() fxevent.Logger {
			return &fxLogrusLogger{}
		}),
		config.Module,
		database.Module,
		repository.Module,
		scraper.Module,   // scraper registry + site adapters
		service.Module,   // site, stats, refresh, notifier-config services
		notifier.Module,  // MultiNotifier + ntfy backend
		scheduler.Module, // cron scheduler (implements domain.Refresher)
		handler.Module,   // HTTP handlers
		server.Module,    // chi router + HTTP server lifecycle
	).Run()
}
