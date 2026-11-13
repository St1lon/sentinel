package main

import (
	"context"
	"errors"
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
	cfg, err := config.LoadProberConfig(config.NewValidator())
	if err != nil {
		return err
	}

	logger := app.NewLogger(&cfg.Log, &cfg.App, app.ServiceProber)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	if err := app.BuildProber(cfg, logger).Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
		return err
	}

	return nil
}
