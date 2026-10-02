// Package createmonitor реализует создание монитора.
package createmonitor

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/St1lon/sentinel/internal/domain"
)

// Usecase — создание монитора.
type Usecase struct {
	monitors MonitorRepo
}

// NewUsecase собирает usecase создания монитора.
func NewUsecase(monitors MonitorRepo) *Usecase {
	return &Usecase{monitors: monitors}
}

// Execute создаёт монитор. next_check_at ставится на «сейчас», поэтому первая
// проверка выполняется на ближайшем цикле воркера, а не через interval.
func (uc *Usecase) Execute(ctx context.Context, req *Request) (*domain.Monitor, error) {
	if err := req.validate(); err != nil {
		return nil, err
	}

	now := time.Now().UTC()

	monitor := &domain.Monitor{
		ID:               uuid.NewString(),
		UserID:           req.UserID,
		Name:             req.Name,
		Kind:             req.kind,
		Target:           req.Target,
		Method:           req.method,
		IntervalSeconds:  req.intervalSeconds,
		TimeoutSeconds:   req.timeoutSeconds,
		ExpectedStatus:   req.expectedStatus,
		FailureThreshold: req.failureThreshold,
		IsPublic:         req.IsPublic,
		Paused:           req.Paused,
		Status:           domain.MonitorStatusPending,
		NextCheckAt:      now,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	if monitor.Paused {
		monitor.Status = domain.MonitorStatusPaused
	}

	if err := uc.monitors.Create(ctx, monitor); err != nil {
		return nil, err
	}

	return monitor, nil
}
