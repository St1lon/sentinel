package listmonitors

import (
	"context"

	"github.com/St1lon/sentinel/internal/domain"
)

type Result struct {
	Monitors []*domain.Monitor
	Total    int
	Limit    int
	Offset   int
}

type Usecase struct {
	monitors MonitorRepo
}

func NewUsecase(monitors MonitorRepo) *Usecase {
	return &Usecase{monitors: monitors}
}

func (uc *Usecase) Execute(ctx context.Context, req *Request) (*Result, error) {
	if err := req.validate(); err != nil {
		return nil, err
	}

	monitors, total, err := uc.monitors.ListByUser(ctx, req.UserID, req.Limit, req.Offset)
	if err != nil {
		return nil, err
	}

	return &Result{Monitors: monitors, Total: total, Limit: req.Limit, Offset: req.Offset}, nil
}
