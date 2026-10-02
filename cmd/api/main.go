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
	cfg, err := config.LoadAPIConfig(config.NewValidator())
	if err != nil {
		return err
	}

	logger := app.NewLogger(&cfg.Log, &cfg.App)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	api, err := app.BuildAPI(ctx, cfg, logger)
	if err != nil {
		return err
	}

	errCh := make(chan error, 1)

	go func() { errCh <- api.Run() }()

	select {
	case err := <-errCh:
		if err != nil {
			return err
		}
	case <-ctx.Done():
		logger.Info("shutdown signal received")
	}

	if err := api.Shutdown(context.WithoutCancel(ctx)); err != nil && !errors.Is(err, context.Canceled) {
		logger.Error("shutdown failed", slog.String("error", err.Error()))

		return err
	}

	return nil
}
