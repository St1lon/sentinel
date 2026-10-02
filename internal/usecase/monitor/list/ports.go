package listmonitors

import (
	"context"

	"github.com/St1lon/sentinel/internal/domain"
)

type MonitorRepo interface {
	ListByUser(ctx context.Context, userID string, limit, offset int) ([]*domain.Monitor, int, error)
}
