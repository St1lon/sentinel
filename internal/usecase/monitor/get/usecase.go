package getmonitor

import (
	"context"

	"github.com/St1lon/sentinel/internal/domain"
	"github.com/St1lon/sentinel/internal/usecase/common/validate"
)

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

type Usecase struct {
	monitors MonitorRepo
}

func NewUsecase(monitors MonitorRepo) *Usecase {
	return &Usecase{monitors: monitors}
}

func (uc *Usecase) Execute(ctx context.Context, req *Request) (*domain.Monitor, error) {
	if err := req.validate(); err != nil {
		return nil, err
	}

	return uc.monitors.GetByIDForUser(ctx, req.MonitorID, req.UserID)
}
