package updatemonitor

import (
	"time"

	"github.com/St1lon/sentinel/internal/domain"
	"github.com/St1lon/sentinel/internal/usecase/common/validate"
)

// Request — частичное обновление: nil означает «поле не передано».
type Request struct {
	UserID    string
	MonitorID string

	Name             *string
	Target           *string
	Method           *string
	IntervalSeconds  *int
	TimeoutSeconds   *int
	ExpectedStatus   *int
	FailureThreshold *int
	IsPublic         *bool
	Paused           *bool
}

// validate проверяет переданные поля. Полную пару интервал/таймаут
// проверить здесь нельзя — не хватает текущих значений, поэтому
// кросс-проверка выполняется в applyTo.
//
//nolint:cyclop // линейная проверка независимых опциональных полей
func (req *Request) validate() error {
	if !validate.UUID(req.UserID) {
		return domain.ErrUnauthenticated
	}

	if !validate.UUID(req.MonitorID) {
		return domain.ErrMonitorNotFound
	}

	if req.isEmpty() {
		return domain.ErrNothingToUpdate
	}

	if req.Name != nil {
		name, err := validate.MonitorName(*req.Name)
		if err != nil {
			return err
		}

		req.Name = &name
	}

	if req.Target != nil {
		target, err := validate.MonitorTarget(*req.Target)
		if err != nil {
			return err
		}

		req.Target = &target
	}

	if req.Method != nil {
		method, err := validate.Method(*req.Method)
		if err != nil {
			return err
		}

		req.Method = &method
	}

	if req.ExpectedStatus != nil {
		if err := validate.ExpectedStatus(*req.ExpectedStatus); err != nil {
			return err
		}
	}

	if req.FailureThreshold != nil {
		if err := validate.FailureThreshold(*req.FailureThreshold); err != nil {
			return err
		}
	}

	return nil
}

func (req *Request) isEmpty() bool {
	return req.Name == nil &&
		req.Target == nil &&
		req.Method == nil &&
		req.IntervalSeconds == nil &&
		req.TimeoutSeconds == nil &&
		req.ExpectedStatus == nil &&
		req.FailureThreshold == nil &&
		req.IsPublic == nil &&
		req.Paused == nil
}

// applyTo переносит переданные поля в монитор и сообщает, изменился ли признак паузы.
func (req *Request) applyTo(monitor *domain.Monitor) bool {
	if req.Name != nil {
		monitor.Name = *req.Name
	}

	if req.Target != nil {
		monitor.Target = *req.Target
	}

	if req.Method != nil {
		monitor.Method = *req.Method
	}

	if req.IntervalSeconds != nil {
		monitor.IntervalSeconds = *req.IntervalSeconds
	}

	if req.TimeoutSeconds != nil {
		monitor.TimeoutSeconds = *req.TimeoutSeconds
	}

	if req.ExpectedStatus != nil {
		monitor.ExpectedStatus = *req.ExpectedStatus
	}

	if req.FailureThreshold != nil {
		monitor.FailureThreshold = *req.FailureThreshold
	}

	if req.IsPublic != nil {
		monitor.IsPublic = *req.IsPublic
	}

	return req.applyPause(monitor)
}

func (req *Request) applyPause(monitor *domain.Monitor) bool {
	if req.Paused == nil || *req.Paused == monitor.Paused {
		return false
	}

	monitor.Paused = *req.Paused

	if monitor.Paused {
		monitor.Status = domain.MonitorStatusPaused

		return true
	}

	// Возврат из паузы: историю падений сбрасываем и проверяем сразу,
	// не дожидаясь следующего интервала.
	monitor.Status = domain.MonitorStatusPending
	monitor.ConsecutiveFailures = 0
	monitor.NextCheckAt = time.Now().UTC()

	return true
}

// ValidateSchedule проверяет пару интервал/таймаут с учётом текущих значений
// монитора: при PATCH может прийти только одно из двух полей.
func (req *Request) ValidateSchedule(monitor *domain.Monitor) error {
	interval := monitor.IntervalSeconds
	if req.IntervalSeconds != nil {
		interval = *req.IntervalSeconds
	}

	timeout := monitor.TimeoutSeconds
	if req.TimeoutSeconds != nil {
		timeout = *req.TimeoutSeconds
	}

	return validate.Schedule(interval, timeout)
}
