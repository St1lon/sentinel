package recordprobe

import (
	"context"
	"time"

	"github.com/St1lon/sentinel/internal/domain"
)

type MonitorRepo interface {
	ApplyProbeResult(
		ctx context.Context,
		monitorID string,
		status domain.MonitorStatus,
		consecutiveFailures int,
		checkedAt time.Time,
	) error
}

type CheckRepo interface {
	Insert(ctx context.Context, check *domain.Check) error
}

type IncidentRepo interface {
	Open(ctx context.Context, incident *domain.Incident) error
	ResolveOpen(ctx context.Context, monitorID string, resolvedAt time.Time) error
}
