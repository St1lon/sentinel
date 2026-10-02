// Package listincidents реализует историю инцидентов монитора.
package listincidents

import (
	"context"

	"github.com/St1lon/sentinel/internal/domain"
	"github.com/St1lon/sentinel/internal/usecase/common/validate"
)

// Request — монитор и параметры страницы.
type Request struct {
	UserID    string
	MonitorID string
	Limit     int
	Offset    int
}

func (req *Request) validate() error {
	if !validate.UUID(req.UserID) {
		return domain.ErrUnauthenticated
	}

	if !validate.UUID(req.MonitorID) {
		return domain.ErrMonitorNotFound
	}

	limit, offset, err := validate.Paging(req.Limit, req.Offset)
	if err != nil {
		return err
	}

	req.Limit, req.Offset = limit, offset

	return nil
}

// Result — страница инцидентов монитора.
type Result struct {
	Monitor   *domain.Monitor
	Incidents []*domain.Incident
	Total     int
	Limit     int
	Offset    int
}

// Usecase — история инцидентов монитора.
type Usecase struct {
	monitors  MonitorRepo
	incidents IncidentRepo
}

// NewUsecase собирает usecase истории инцидентов.
func NewUsecase(monitors MonitorRepo, incidents IncidentRepo) *Usecase {
	return &Usecase{monitors: monitors, incidents: incidents}
}

// Execute возвращает инциденты монитора пользователя.
func (uc *Usecase) Execute(ctx context.Context, req *Request) (*Result, error) {
	if err := req.validate(); err != nil {
		return nil, err
	}

	monitor, err := uc.monitors.GetByIDForUser(ctx, req.MonitorID, req.UserID)
	if err != nil {
		return nil, err
	}

	incidents, total, err := uc.incidents.ListByMonitor(ctx, monitor.ID, req.Limit, req.Offset)
	if err != nil {
		return nil, err
	}

	return &Result{
		Monitor:   monitor,
		Incidents: incidents,
		Total:     total,
		Limit:     req.Limit,
		Offset:    req.Offset,
	}, nil
}
