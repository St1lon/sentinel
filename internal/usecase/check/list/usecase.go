// Package listchecks реализует чтение последних проверок монитора.
package listchecks

import (
	"context"
	"time"

	"github.com/St1lon/sentinel/internal/domain"
	"github.com/St1lon/sentinel/internal/usecase/common/validate"
)

// Request — монитор, период и ограничение выборки.
type Request struct {
	UserID    string
	MonitorID string
	From      time.Time
	To        time.Time
	Limit     int
}

func (req *Request) validate() error {
	if !validate.UUID(req.UserID) {
		return domain.ErrUnauthenticated
	}

	if !validate.UUID(req.MonitorID) {
		return domain.ErrMonitorNotFound
	}

	if err := validate.TimeRange(req.From, req.To); err != nil {
		return err
	}

	limit, _, err := validate.Paging(req.Limit, 0)
	if err != nil {
		return err
	}

	req.Limit = limit

	return nil
}

// Result — проверки монитора, свежие сверху.
type Result struct {
	Monitor *domain.Monitor
	Checks  []*domain.Check
}

// Usecase — чтение проверок монитора.
type Usecase struct {
	monitors MonitorRepo
	checks   CheckRepo
}

// NewUsecase собирает usecase чтения проверок.
func NewUsecase(monitors MonitorRepo, checks CheckRepo) *Usecase {
	return &Usecase{monitors: monitors, checks: checks}
}

// Execute возвращает проверки монитора пользователя за период.
func (uc *Usecase) Execute(ctx context.Context, req *Request) (*Result, error) {
	if err := req.validate(); err != nil {
		return nil, err
	}

	monitor, err := uc.monitors.GetByIDForUser(ctx, req.MonitorID, req.UserID)
	if err != nil {
		return nil, err
	}

	checks, err := uc.checks.ListByMonitor(ctx, monitor.ID, req.From, req.To, req.Limit)
	if err != nil {
		return nil, err
	}

	return &Result{Monitor: monitor, Checks: checks}, nil
}
