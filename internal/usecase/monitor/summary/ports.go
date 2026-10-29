package monitorsummary

import (
	"context"

	"github.com/St1lon/sentinel/internal/domain"
)

type MonitorRepo interface {
	CountByStatus(ctx context.Context) (map[domain.MonitorStatus]int, error)
}

type IncidentRepo interface {
	CountOpen(ctx context.Context) (int, error)
}
