package listmonitors

import (
	"context"

	"github.com/St1lon/sentinel/internal/domain"
)

// MonitorRepo — порт постраничного чтения мониторов пользователя.
type MonitorRepo interface {
	ListByUser(ctx context.Context, userID string, limit, offset int) ([]*domain.Monitor, int, error)
}
