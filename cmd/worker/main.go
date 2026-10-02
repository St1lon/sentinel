// Команда worker — процесс проверок приложения Sentinel.
package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/St1lon/sentinel/internal/app"
	"github.com/St1lon/sentinel/internal/config"
)

func main() {
	if err := run(); err != nil {
		_, _ = os.Stderr.WriteString("fatal: " + err.Error() + "\n")
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.LoadWorkerConfig(config.NewValidator())
	if err != nil {
		return err
	}

	logger := app.NewLogger(&cfg.Log, &cfg.App)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	probeWorker, err := app.BuildWorker(ctx, cfg, logger)
	if err != nil {
		return err
	}

	runErr := probeWorker.Run(ctx)

	if err := probeWorker.Shutdown(context.WithoutCancel(ctx)); err != nil {
		logger.Error("shutdown failed", slog.String("error", err.Error()))
	}

	if runErr != nil && !errors.Is(runErr, context.Canceled) {
		return runErr
	}

	return nil
}
