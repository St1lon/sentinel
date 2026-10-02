package updatemonitor

import (
	"context"
	"time"

	"github.com/St1lon/sentinel/internal/domain"
)

// MonitorRepo — порт чтения и записи монитора.
type MonitorRepo interface {
	GetByIDForUser(ctx context.Context, id, userID string) (*domain.Monitor, error)
	Update(ctx context.Context, monitor *domain.Monitor) error
}

// IncidentRepo — порт закрытия открытого инцидента при постановке на паузу.
type IncidentRepo interface {
	ResolveOpen(ctx context.Context, monitorID string, resolvedAt time.Time) error
}
