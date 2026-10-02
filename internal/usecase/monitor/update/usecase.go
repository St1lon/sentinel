// Package updatemonitor реализует частичное обновление монитора.
package updatemonitor

import (
	"context"
	"time"

	"github.com/St1lon/sentinel/internal/domain"
	"github.com/St1lon/sentinel/internal/usecase/common"
)

// Usecase — частичное обновление монитора.
type Usecase struct {
	monitors  MonitorRepo
	incidents IncidentRepo
	tx        common.TxManager
}

// NewUsecase собирает usecase обновления монитора.
func NewUsecase(monitors MonitorRepo, incidents IncidentRepo, tx common.TxManager) *Usecase {
	return &Usecase{monitors: monitors, incidents: incidents, tx: tx}
}

// Execute применяет переданные поля к монитору.
//
// Чтение, изменение и запись идут в одной транзакции: иначе два одновременных
// PATCH'а могли бы затереть изменения друг друга (lost update).
// Снятие монитора с паузы дополнительно закрывает открытый инцидент — это
// вторая операция в той же транзакции, поэтому нужен TxManager.
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

		// Пара интервал/таймаут проверяется здесь: при PATCH могло прийти
		// только одно из полей, и второе берётся из текущего состояния.
		if err := req.ValidateSchedule(monitor); err != nil {
			return err
		}

		pauseChanged := req.applyTo(monitor)
		monitor.UpdatedAt = time.Now().UTC()

		if err := uc.monitors.Update(ctx, monitor); err != nil {
			return err
		}

		// Монитор поставили на паузу — висящий инцидент надо закрыть,
		// иначе статус-страница навсегда останется красной.
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
