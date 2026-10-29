package monitorsummary

import (
	"context"

	"github.com/St1lon/sentinel/internal/domain"
)

// Usecase собирает срез состояния всех мониторов: по одному запросу
// в мониторы и в инциденты.
type Usecase struct {
	monitors  MonitorRepo
	incidents IncidentRepo
}

func NewUsecase(monitors MonitorRepo, incidents IncidentRepo) *Usecase {
	return &Usecase{monitors: monitors, incidents: incidents}
}

func (uc *Usecase) Execute(ctx context.Context) (*domain.FleetSummary, error) {
	byStatus, err := uc.monitors.CountByStatus(ctx)
	if err != nil {
		return nil, err
	}

	openIncidents, err := uc.incidents.CountOpen(ctx)
	if err != nil {
		return nil, err
	}

	return &domain.FleetSummary{ByStatus: byStatus, OpenIncidents: openIncidents}, nil
}
