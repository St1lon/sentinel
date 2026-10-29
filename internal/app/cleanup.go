package app

import (
	"context"
	"log/slog"
	"time"

	"github.com/St1lon/sentinel/internal/config"
	"github.com/St1lon/sentinel/internal/infra/db/postgres"
	"github.com/St1lon/sentinel/internal/infra/db/postgres/repository"
	"github.com/St1lon/sentinel/internal/infra/metrics"
	cleanupchecks "github.com/St1lon/sentinel/internal/usecase/maintenance/cleanup"
)

func RunCleanup(ctx context.Context, cfg *config.CleanupConfig, logger *slog.Logger) error {
	pool, err := postgres.NewPool(ctx, &cfg.Postgres)
	if err != nil {
		return err
	}
	defer pool.Close()

	registry := metrics.NewJobRegistry(ServiceCleanup)
	cleanupMetrics := metrics.NewCleanup(registry.Registerer())

	usecase := cleanupchecks.NewUsecase(repository.NewCheckRepo(pool))

	started := time.Now()

	result, err := usecase.Execute(ctx, &cleanupchecks.Request{
		RetentionDays: cfg.Retention.CheckRetentionDays,
		BatchSize:     cfg.Retention.BatchSize,
		Now:           started.UTC(),
	})
	if err != nil {
		return err
	}

	finished := time.Now()

	logger.Info("cleanup finished",
		slog.Int64("deleted_checks", result.Deleted),
		slog.Time("older_than", result.Before),
		slog.Int("retention_days", cfg.Retention.CheckRetentionDays),
	)

	cleanupMetrics.Finished(result.Deleted, finished.Sub(started), finished.UTC())
	pushMetrics(ctx, &cfg.Push, registry, logger)

	return nil
}

// pushMetrics отдаёт метрики задачи push-моделью. Неудачная отправка не
// роняет очистку: записи уже удалены, повторять задачу из-за мониторинга
// незачем.
func pushMetrics(
	ctx context.Context, cfg *config.MetricsPushConfig, registry *metrics.Registry, logger *slog.Logger,
) {
	if cfg.URL == "" {
		logger.Debug("metrics push disabled: METRICS_PUSH_URL is empty")

		return
	}

	pusher, err := metrics.NewPusher(cfg, ServiceCleanup)
	if err != nil {
		logger.Error("metrics push setup failed", slog.String("error", err.Error()))

		return
	}

	if err := pusher.Push(ctx, registry.Gatherer()); err != nil {
		logger.Error("metrics push failed", slog.String("error", err.Error()))

		return
	}

	logger.Info("metrics pushed", slog.String("url", cfg.URL))
}
