// Команда cleanup — одноразовый административный процесс: удаляет проверки
// старше заданного срока хранения и завершается.
//
// Запускается тем же образом и с тем же окружением, что api и worker:
// `docker compose run --rm cleanup` или как CronJob в кластере.
package main

import (
	"context"
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
	cfg, err := config.LoadCleanupConfig(config.NewValidator())
	if err != nil {
		return err
	}

	logger := app.NewLogger(&cfg.Log, &cfg.App)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	return app.RunCleanup(ctx, cfg, logger)
}
