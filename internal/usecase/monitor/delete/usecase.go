// Package deletemonitor реализует удаление монитора.
package deletemonitor

import (
	"context"

	"github.com/St1lon/sentinel/internal/domain"
	"github.com/St1lon/sentinel/internal/usecase/common/validate"
)

// Request — идентификаторы удаляемого монитора и его владельца.
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

// Usecase — удаление монитора.
type Usecase struct {
	monitors MonitorRepo
}

// NewUsecase собирает usecase удаления монитора.
func NewUsecase(monitors MonitorRepo) *Usecase {
	return &Usecase{monitors: monitors}
}

// Execute удаляет монитор пользователя. Проверки и инциденты удаляются
// каскадом на уровне схемы БД.
func (uc *Usecase) Execute(ctx context.Context, req *Request) error {
	if err := req.validate(); err != nil {
		return err
	}

	return uc.monitors.Delete(ctx, req.MonitorID, req.UserID)
}
