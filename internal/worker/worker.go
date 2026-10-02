// Package worker — процесс проверок. Он забирает у БД мониторы, которым пора
// проверяться, параллельно их пингует и отдаёт результаты в usecase.
// Сама бизнес-логика (когда монитор считается упавшим) живёт в domain,
// здесь только оркестрация и параллелизм.
package worker

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/St1lon/sentinel/internal/domain"
	leasemonitors "github.com/St1lon/sentinel/internal/usecase/probe/lease"
	recordprobe "github.com/St1lon/sentinel/internal/usecase/probe/record"
)

// Prober — порт выполнения одной проверки; реализуется infra/prober.
type Prober interface {
	Probe(ctx context.Context, monitor *domain.Monitor) (*domain.Check, error)
}

// Options — параметры цикла проверок.
type Options struct {
	Concurrency  int
	BatchSize    int
	PollInterval time.Duration
}

// Worker — цикл проверок.
type Worker struct {
	lease  *leasemonitors.Usecase
	record *recordprobe.Usecase
	prober Prober
	logger *slog.Logger
	opts   Options
}

// New собирает воркер.
func New(
	lease *leasemonitors.Usecase,
	record *recordprobe.Usecase,
	prober Prober,
	logger *slog.Logger,
	opts Options,
) *Worker {
	return &Worker{lease: lease, record: record, prober: prober, logger: logger, opts: opts}
}

// Run крутит цикл до отмены контекста.
//
// Воркер не хранит состояние в памяти: расписание живёт в БД, поэтому его можно
// убить в любой момент и запустить сколько угодно экземпляров (факторы VI и IX).
func (w *Worker) Run(ctx context.Context) error {
	w.logger.Info("worker started",
		slog.Int("concurrency", w.opts.Concurrency),
		slog.Int("batch_size", w.opts.BatchSize),
		slog.Duration("poll_interval", w.opts.PollInterval),
	)

	ticker := time.NewTicker(w.opts.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			w.logger.Info("worker stopped")

			return nil
		case <-ticker.C:
			if err := w.runOnce(ctx); err != nil && !errors.Is(err, context.Canceled) {
				// Ошибка цикла не роняет процесс: следующий тик попробует снова.
				w.logger.ErrorContext(ctx, "probe cycle failed", slog.String("error", err.Error()))
			}
		}
	}
}

// runOnce обрабатывает одну пачку мониторов.
func (w *Worker) runOnce(ctx context.Context) error {
	monitors, err := w.lease.Execute(ctx, &leasemonitors.Request{
		BatchSize: w.opts.BatchSize,
		Now:       time.Now().UTC(),
	})
	if err != nil {
		return err
	}

	if len(monitors) == 0 {
		return nil
	}

	group, groupCtx := errgroup.WithContext(ctx)
	group.SetLimit(w.opts.Concurrency)

	for _, monitor := range monitors {
		group.Go(func() error {
			w.checkOne(groupCtx, monitor)

			return nil
		})
	}

	if err := group.Wait(); err != nil {
		return err
	}

	w.logger.DebugContext(ctx, "probe cycle finished", slog.Int("monitors", len(monitors)))

	return nil
}

// checkOne проверяет один монитор и записывает результат.
// Ошибка по одному монитору не должна отменять остальную пачку, поэтому она
// логируется, а не возвращается в errgroup.
func (w *Worker) checkOne(ctx context.Context, monitor *domain.Monitor) {
	check, err := w.prober.Probe(ctx, monitor)
	if err != nil {
		w.logger.WarnContext(ctx, "probe skipped",
			slog.String("monitor_id", monitor.ID),
			slog.String("kind", string(monitor.Kind)),
			slog.String("error", err.Error()),
		)

		return
	}

	result, err := w.record.Execute(ctx, &recordprobe.Request{Monitor: monitor, Check: check})
	if err != nil {
		if !errors.Is(err, context.Canceled) {
			w.logger.ErrorContext(ctx, "record probe failed",
				slog.String("monitor_id", monitor.ID),
				slog.String("error", err.Error()),
			)
		}

		return
	}

	w.logStateChange(ctx, monitor, check, result)
}

func (w *Worker) logStateChange(
	ctx context.Context, monitor *domain.Monitor, check *domain.Check, result *recordprobe.Result,
) {
	if !result.Transition.StatusChanged {
		return
	}

	attrs := []slog.Attr{
		slog.String("monitor_id", monitor.ID),
		slog.String("monitor_name", monitor.Name),
		slog.String("status", string(result.Transition.Status)),
		slog.Int("latency_ms", check.LatencyMS),
	}

	if result.Transition.OpenIncident {
		attrs = append(attrs, slog.String("cause", result.Transition.Cause))
		w.logger.LogAttrs(ctx, slog.LevelWarn, "incident opened", attrs...)

		return
	}

	if result.Transition.CloseIncident {
		w.logger.LogAttrs(ctx, slog.LevelInfo, "incident resolved", attrs...)

		return
	}

	w.logger.LogAttrs(ctx, slog.LevelInfo, "monitor status changed", attrs...)
}
