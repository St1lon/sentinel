// Package getmonitor реализует чтение одного монитора.
package getmonitor

import (
	"context"

	"github.com/St1lon/sentinel/internal/domain"
	"github.com/St1lon/sentinel/internal/usecase/common/validate"
)

// Request — идентификаторы монитора и его владельца.
type Request struct {
	UserID    string
	MonitorID string
}

func (req *Request) validate() error {
	if !validate.UUID(req.UserID) {
		return domain.ErrUnauthenticated
	}

	if !validate.UUID(req.MonitorID) {
		return domain.ErrMonitorNotFound
	}

	return nil
}

// Usecase — чтение монитора.
type Usecase struct {
	monitors MonitorRepo
}

// NewUsecase собирает usecase чтения монитора.
func NewUsecase(monitors MonitorRepo) *Usecase {
	return &Usecase{monitors: monitors}
}

// Execute возвращает монитор пользователя.
func (uc *Usecase) Execute(ctx context.Context, req *Request) (*domain.Monitor, error) {
	if err := req.validate(); err != nil {
		return nil, err
	}

	return uc.monitors.GetByIDForUser(ctx, req.MonitorID, req.UserID)
}
