package app

import (
	"context"
	"fmt"
	"log/slog"

	"golang.org/x/sync/errgroup"

	"github.com/St1lon/sentinel/internal/blackbox"
	"github.com/St1lon/sentinel/internal/config"
	"github.com/St1lon/sentinel/internal/infra/metrics"
	transportmetrics "github.com/St1lon/sentinel/internal/transport/metrics"
)

type Prober struct {
	journey       *blackbox.Journey
	metricsServer *transportmetrics.Server
	logger        *slog.Logger
}

// BuildProber не принимает контекст и не возвращает ошибку: пробер не ходит
// в базу, и собирать ему нечего, кроме HTTP-клиента и реестра метрик.
func BuildProber(cfg *config.ProberProcessConfig, logger *slog.Logger) *Prober {
	registry := metrics.NewRegistry(ServiceProber)

	journey := blackbox.New(
		metrics.NewSLI(registry.Registerer()),
		logger,
		blackbox.Options{
			TargetURL:     cfg.Prober.TargetURL,
			Interval:      cfg.Prober.Interval,
			Timeout:       cfg.Prober.Timeout,
			LoginEmail:    cfg.Prober.LoginEmail,
			LoginPassword: cfg.Prober.LoginPassword,
			StatusSlug:    cfg.Prober.StatusSlug,
			UserAgent:     cfg.Prober.UserAgent,
		},
	)

	return &Prober{
		journey:       journey,
		metricsServer: transportmetrics.NewServer(&cfg.Metrics, registry.Handler(), logger),
		logger:        logger,
	}
}

// Run держит две горутины: прогон сценария по тикеру и порт сбора метрик.
// Последний сам по контексту не останавливается, поэтому его гасит
// отдельная горутина.
func (p *Prober) Run(ctx context.Context) error {
	group, groupCtx := errgroup.WithContext(ctx)

	group.Go(func() error { return p.journey.Run(groupCtx) })
	group.Go(p.metricsServer.Run)

	group.Go(func() error {
		<-groupCtx.Done()

		return p.metricsServer.Shutdown(context.WithoutCancel(groupCtx))
	})

	if err := group.Wait(); err != nil {
		return fmt.Errorf("run prober: %w", err)
	}

	return nil
}
