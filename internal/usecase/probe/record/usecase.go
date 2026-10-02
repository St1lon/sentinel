// Package recordprobe фиксирует результат одной проверки монитора.
package recordprobe

import (
	"context"

	"github.com/google/uuid"

	"github.com/St1lon/sentinel/internal/domain"
	"github.com/St1lon/sentinel/internal/usecase/common"
)

// Result — что именно изменилось: нужно воркеру для логов и будущих алертов.
type Result struct {
	Transition domain.Transition
	CheckID    int64
}

// Usecase — запись результата проверки и применение перехода состояния.
type Usecase struct {
	monitors  MonitorRepo
	checks    CheckRepo
	incidents IncidentRepo
	tx        common.TxManager
}

// NewUsecase собирает usecase записи результата проверки.
func NewUsecase(monitors MonitorRepo, checks CheckRepo, incidents IncidentRepo, tx common.TxManager) *Usecase {
	return &Usecase{monitors: monitors, checks: checks, incidents: incidents, tx: tx}
}

// Execute записывает проверку и применяет вычисленный доменом переход.
//
// Все четыре записи (проверка, статус монитора, открытие и закрытие инцидента)
// идут в одной транзакции: иначе падение процесса между ними оставило бы
// монитор в статусе down без инцидента или наоборот.
func (uc *Usecase) Execute(ctx context.Context, req *Request) (*Result, error) {
	if err := req.validate(); err != nil {
		return nil, err
	}

	transition := domain.EvaluateProbe(req.Monitor, req.Check)
	result := &Result{Transition: transition}

	err := uc.tx.WithTx(ctx, func(ctx context.Context) error {
		if err := uc.checks.Insert(ctx, req.Check); err != nil {
			return err
		}

		result.CheckID = req.Check.ID

		err := uc.monitors.ApplyProbeResult(
			ctx, req.Monitor.ID, transition.Status, transition.ConsecutiveFailures, req.Check.CheckedAt,
		)
		if err != nil {
			return err
		}

		if transition.OpenIncident {
			incident := &domain.Incident{
				ID:        uuid.NewString(),
				MonitorID: req.Monitor.ID,
				StartedAt: req.Check.CheckedAt,
				Cause:     transition.Cause,
			}

			if err := uc.incidents.Open(ctx, incident); err != nil {
				return err
			}
		}

		if transition.CloseIncident {
			if err := uc.incidents.ResolveOpen(ctx, req.Monitor.ID, req.Check.CheckedAt); err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}
