package app

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/sync/errgroup"

	"github.com/St1lon/sentinel/internal/config"
	"github.com/St1lon/sentinel/internal/infra/db/postgres"
	"github.com/St1lon/sentinel/internal/infra/db/postgres/repository"
	"github.com/St1lon/sentinel/internal/infra/metrics"
	"github.com/St1lon/sentinel/internal/infra/prober"
	transportmetrics "github.com/St1lon/sentinel/internal/transport/metrics"
	monitorsummary "github.com/St1lon/sentinel/internal/usecase/monitor/summary"
	leasemonitors "github.com/St1lon/sentinel/internal/usecase/probe/lease"
	recordprobe "github.com/St1lon/sentinel/internal/usecase/probe/record"
	"github.com/St1lon/sentinel/internal/worker"
)

type Worker struct {
	worker        *worker.Worker
	reporter      *worker.SummaryReporter
	metricsServer *transportmetrics.Server
	pool          *pgxpool.Pool
	logger        *slog.Logger
}

func BuildWorker(ctx context.Context, cfg *config.WorkerProcessConfig, logger *slog.Logger) (*Worker, error) {
	pool, err := postgres.NewPool(ctx, &cfg.Postgres)
	if err != nil {
		return nil, err
	}

	registry := metrics.NewRegistry(ServiceWorker)
	txManager := postgres.NewTxManager(pool)

	monitors := repository.NewMonitorRepo(pool)
	checks := repository.NewCheckRepo(pool)
	incidents := repository.NewIncidentRepo(pool)

	httpProber := prober.NewHTTPProber(prober.Options{
		UserAgent:        cfg.Worker.UserAgent,
		MaxResponseBytes: cfg.Worker.MaxResponseBytes,
		MaxRedirects:     cfg.Worker.MaxRedirects,
		AllowPrivate:     cfg.Targets.AllowPrivate,
	})

	probeWorker := worker.New(
		leasemonitors.NewUsecase(monitors),
		recordprobe.NewUsecase(monitors, checks, incidents, txManager),
		httpProber,
		metrics.NewProbes(registry.Registerer()),
		logger,
		worker.Options{
			Concurrency:  cfg.Worker.Concurrency,
			BatchSize:    cfg.Worker.BatchSize,
			PollInterval: cfg.Worker.PollInterval,
		},
	)

	reporter := worker.NewSummaryReporter(
		monitorsummary.NewUsecase(monitors, incidents),
		metrics.NewFleet(registry.Registerer()),
		logger,
		cfg.Worker.StatsInterval,
	)

	return &Worker{
		worker:        probeWorker,
		reporter:      reporter,
		metricsServer: transportmetrics.NewServer(&cfg.Metrics, registry.Handler(), logger),
		pool:          pool,
		logger:        logger,
	}, nil
}

// Run держит три горутины: цикл проверок, снятие среза состояния мониторов
// и порт сбора метрик. Последний сам не останавливается по контексту,
// поэтому его гасит отдельная горутина.
func (w *Worker) Run(ctx context.Context) error {
	group, groupCtx := errgroup.WithContext(ctx)

	group.Go(func() error { return w.worker.Run(groupCtx) })
	group.Go(func() error { return w.reporter.Run(groupCtx) })
	group.Go(w.metricsServer.Run)

	group.Go(func() error {
		<-groupCtx.Done()

		return w.metricsServer.Shutdown(context.WithoutCancel(groupCtx))
	})

	if err := group.Wait(); err != nil {
		return fmt.Errorf("run worker: %w", err)
	}

	return nil
}

func (w *Worker) Shutdown(_ context.Context) error {
	w.pool.Close()
	w.logger.Info("database pool closed")

	return nil
}
