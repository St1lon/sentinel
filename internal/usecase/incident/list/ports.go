package listincidents

import (
	"context"

	"github.com/St1lon/sentinel/internal/domain"
)

// MonitorRepo — порт чтения монитора с проверкой владельца.
type MonitorRepo interface {
	GetByIDForUser(ctx context.Context, id, userID string) (*domain.Monitor, error)
}

// IncidentRepo — порт постраничного чтения инцидентов монитора.
type IncidentRepo interface {
	ListByMonitor(ctx context.Context, monitorID string, limit, offset int) ([]*domain.Incident, int, error)
}
