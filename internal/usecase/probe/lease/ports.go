package leasemonitors

import (
	"context"
	"time"

	"github.com/St1lon/sentinel/internal/domain"
)

type MonitorRepo interface {
	LeaseDue(ctx context.Context, now time.Time, limit int) ([]*domain.Monitor, error)
}
