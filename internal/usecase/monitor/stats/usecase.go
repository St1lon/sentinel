package monitorstats

import (
	"context"

	"github.com/St1lon/sentinel/internal/domain"
)

type Result struct {
	Monitor *domain.Monitor
	Stats   *domain.MonitorStats
	Buckets []*domain.Bucket
}

type Usecase struct {
	monitors MonitorRepo
	checks   CheckRepo
}

func NewUsecase(monitors MonitorRepo, checks CheckRepo) *Usecase {
	return &Usecase{monitors: monitors, checks: checks}
}

func (uc *Usecase) Execute(ctx context.Context, req *Request) (*Result, error) {
	if err := req.validate(); err != nil {
		return nil, err
	}

	monitor, err := uc.monitors.GetByIDForUser(ctx, req.MonitorID, req.UserID)
	if err != nil {
		return nil, err
	}

	stats, err := uc.checks.Stats(ctx, monitor.ID, req.From, req.To)
	if err != nil {
		return nil, err
	}

	buckets, err := uc.checks.Buckets(ctx, monitor.ID, req.From, req.To, req.bucketSize)
	if err != nil {
		return nil, err
	}

	return &Result{Monitor: monitor, Stats: stats, Buckets: buckets}, nil
}
