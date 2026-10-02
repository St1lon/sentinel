package leasemonitors

import (
	"context"
	"time"

	"github.com/St1lon/sentinel/internal/domain"
)

type Request struct {
	BatchSize int
	Now       time.Time
}

func (req *Request) validate() error {
	if req.BatchSize <= 0 {
		return domain.ErrInvalidPaging
	}

	if req.Now.IsZero() {
		req.Now = time.Now().UTC()
	}

	return nil
}

type Usecase struct {
	monitors MonitorRepo
}

func NewUsecase(monitors MonitorRepo) *Usecase {
	return &Usecase{monitors: monitors}
}

func (uc *Usecase) Execute(ctx context.Context, req *Request) ([]*domain.Monitor, error) {
	if err := req.validate(); err != nil {
		return nil, err
	}

	return uc.monitors.LeaseDue(ctx, req.Now, req.BatchSize)
}
