package updatemonitor

import (
	"context"
	"time"

	"github.com/St1lon/sentinel/internal/domain"
	"github.com/St1lon/sentinel/internal/usecase/common"
)

type Usecase struct {
	monitors  MonitorRepo
	incidents IncidentRepo
	tx        common.TxManager
}

func NewUsecase(monitors MonitorRepo, incidents IncidentRepo, tx common.TxManager) *Usecase {
	return &Usecase{monitors: monitors, incidents: incidents, tx: tx}
}

func (uc *Usecase) Execute(ctx context.Context, req *Request) (*domain.Monitor, error) {
	if err := req.validate(); err != nil {
		return nil, err
	}

	var updated *domain.Monitor

	err := uc.tx.WithTx(ctx, func(ctx context.Context) error {
		monitor, err := uc.monitors.GetByIDForUser(ctx, req.MonitorID, req.UserID)
		if err != nil {
			return err
		}

		if err := req.ValidateSchedule(monitor); err != nil {
			return err
		}

		pauseChanged := req.applyTo(monitor)
		monitor.UpdatedAt = time.Now().UTC()

		if err := uc.monitors.Update(ctx, monitor); err != nil {
			return err
		}

		if pauseChanged && monitor.Paused {
			if err := uc.incidents.ResolveOpen(ctx, monitor.ID, monitor.UpdatedAt); err != nil {
				return err
			}
		}

		updated = monitor

		return nil
	})
	if err != nil {
		return nil, err
	}

	return updated, nil
}
