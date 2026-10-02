package app

import (
	"context"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/St1lon/sentinel/internal/config"
	"github.com/St1lon/sentinel/internal/infra/db/postgres"
	"github.com/St1lon/sentinel/internal/infra/db/postgres/repository"
	"github.com/St1lon/sentinel/internal/infra/prober"
	leasemonitors "github.com/St1lon/sentinel/internal/usecase/probe/lease"
	recordprobe "github.com/St1lon/sentinel/internal/usecase/probe/record"
	"github.com/St1lon/sentinel/internal/worker"
)

// Worker — процесс проверок со всеми его зависимостями.
type Worker struct {
	worker *worker.Worker
	pool   *pgxpool.Pool
	logger *slog.Logger
}

// BuildWorker собирает процесс worker.
//
// Это второй тип процесса из одной и той же кодовой базы: api обслуживает
// запросы, worker выполняет проверки. Масштабируются они независимо
// (фактор VIII «Concurrency»).
func BuildWorker(ctx context.Context, cfg *config.WorkerProcessConfig, logger *slog.Logger) (*Worker, error) {
	pool, err := postgres.NewPool(ctx, &cfg.Postgres)
	if err != nil {
		return nil, err
	}

	txManager := postgres.NewTxManager(pool)

	monitors := repository.NewMonitorRepo(pool)
	checks := repository.NewCheckRepo(pool)
	incidents := repository.NewIncidentRepo(pool)

	httpProber := prober.NewHTTPProber(prober.Options{
		UserAgent:        cfg.Worker.UserAgent,
		MaxResponseBytes: cfg.Worker.MaxResponseBytes,
		MaxRedirects:     cfg.Worker.MaxRedirects,
		AllowPrivate:     cfg.Worker.AllowPrivateTargets,
	})

	probeWorker := worker.New(
		leasemonitors.NewUsecase(monitors),
		recordprobe.NewUsecase(monitors, checks, incidents, txManager),
		httpProber,
		logger,
		worker.Options{
			Concurrency:  cfg.Worker.Concurrency,
			BatchSize:    cfg.Worker.BatchSize,
			PollInterval: cfg.Worker.PollInterval,
		},
	)

	return &Worker{worker: probeWorker, pool: pool, logger: logger}, nil
}

// Run крутит цикл проверок до отмены контекста.
func (w *Worker) Run(ctx context.Context) error {
	return w.worker.Run(ctx)
}

// Shutdown закрывает пул соединений.
func (w *Worker) Shutdown(_ context.Context) error {
	w.pool.Close()
	w.logger.Info("database pool closed")

	return nil
}
