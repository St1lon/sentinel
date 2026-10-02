package recordprobe

import (
	"context"

	"github.com/google/uuid"

	"github.com/St1lon/sentinel/internal/domain"
	"github.com/St1lon/sentinel/internal/usecase/common"
)

type Result struct {
	Transition domain.Transition
	CheckID    int64
}

type Usecase struct {
	monitors  MonitorRepo
	checks    CheckRepo
	incidents IncidentRepo
	tx        common.TxManager
}

func NewUsecase(monitors MonitorRepo, checks CheckRepo, incidents IncidentRepo, tx common.TxManager) *Usecase {
	return &Usecase{monitors: monitors, checks: checks, incidents: incidents, tx: tx}
}

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
