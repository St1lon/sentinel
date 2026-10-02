// Команда api — HTTP-процесс приложения Sentinel.
// main делает ровно три вещи: читает конфигурацию, собирает приложение
// и управляет его завершением. Вся логика сборки — в internal/app.
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
		// Логгера может ещё не быть (ошибка конфигурации), поэтому stderr.
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

	// SIGTERM — штатный сигнал остановки контейнера; по нему начинается
	// graceful shutdown, а не немедленная смерть процесса (фактор IX).
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
