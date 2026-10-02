package app

import (
	"context"
	"log/slog"
	"time"

	"github.com/St1lon/sentinel/internal/config"
	"github.com/St1lon/sentinel/internal/infra/db/postgres"
	"github.com/St1lon/sentinel/internal/infra/db/postgres/repository"
	cleanupchecks "github.com/St1lon/sentinel/internal/usecase/maintenance/cleanup"
)

func RunCleanup(ctx context.Context, cfg *config.CleanupConfig, logger *slog.Logger) error {
	pool, err := postgres.NewPool(ctx, &cfg.Postgres)
	if err != nil {
		return err
	}
	defer pool.Close()

	usecase := cleanupchecks.NewUsecase(repository.NewCheckRepo(pool))

	result, err := usecase.Execute(ctx, &cleanupchecks.Request{
		RetentionDays: cfg.Retention.CheckRetentionDays,
		BatchSize:     cfg.Retention.BatchSize,
		Now:           time.Now().UTC(),
	})
	if err != nil {
		return err
	}

	logger.Info("cleanup finished",
		slog.Int64("deleted_checks", result.Deleted),
		slog.Time("older_than", result.Before),
		slog.Int("retention_days", cfg.Retention.CheckRetentionDays),
	)

	return nil
}
