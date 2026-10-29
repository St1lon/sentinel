package worker

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/St1lon/sentinel/internal/domain"
	monitorsummary "github.com/St1lon/sentinel/internal/usecase/monitor/summary"
)

// FleetMetrics — gauge-метрики состояния мониторов.
type FleetMetrics interface {
	SetFleetSummary(summary *domain.FleetSummary)
}

// SummaryReporter периодически снимает срез состояния мониторов и обновляет
// по нему gauge-метрики. Срез снимает воркер, а не API: это одно значение
// на всю систему, а не на реплику, и запрос с group by не должен выполняться
// в момент опроса метрик.
type SummaryReporter struct {
	summary  *monitorsummary.Usecase
	metrics  FleetMetrics
	logger   *slog.Logger
	interval time.Duration
}

func NewSummaryReporter(
	summary *monitorsummary.Usecase,
	metrics FleetMetrics,
	logger *slog.Logger,
	interval time.Duration,
) *SummaryReporter {
	return &SummaryReporter{summary: summary, metrics: metrics, logger: logger, interval: interval}
}

func (r *SummaryReporter) Run(ctx context.Context) error {
	r.logger.Info("fleet summary reporter started", slog.Duration("interval", r.interval))

	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	// Первый срез — сразу: иначе до первого тика gauge-метрик в выдаче нет.
	r.report(ctx)

	for {
		select {
		case <-ctx.Done():
			r.logger.Info("fleet summary reporter stopped")

			return nil
		case <-ticker.C:
			r.report(ctx)
		}
	}
}

func (r *SummaryReporter) report(ctx context.Context) {
	summary, err := r.summary.Execute(ctx)
	if err != nil {
		if !errors.Is(err, context.Canceled) {
			r.logger.ErrorContext(ctx, "fleet summary failed", slog.String("error", err.Error()))
		}

		return
	}

	r.metrics.SetFleetSummary(summary)
}
