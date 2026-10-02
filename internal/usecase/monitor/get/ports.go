package getmonitor

import (
	"context"

	"github.com/St1lon/sentinel/internal/domain"
)

// MonitorRepo — порт чтения монитора с проверкой владельца.
type MonitorRepo interface {
	GetByIDForUser(ctx context.Context, id, userID string) (*domain.Monitor, error)
}
