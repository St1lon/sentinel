package recordprobe

import (
	"context"
	"time"

	"github.com/St1lon/sentinel/internal/domain"
)

// MonitorRepo — порт применения итога проверки к монитору.
type MonitorRepo interface {
	ApplyProbeResult(
		ctx context.Context,
		monitorID string,
		status domain.MonitorStatus,
		consecutiveFailures int,
		checkedAt time.Time,
	) error
}

// CheckRepo — порт записи результата проверки.
type CheckRepo interface {
	Insert(ctx context.Context, check *domain.Check) error
}

// IncidentRepo — порт открытия и закрытия инцидентов.
type IncidentRepo interface {
	Open(ctx context.Context, incident *domain.Incident) error
	ResolveOpen(ctx context.Context, monitorID string, resolvedAt time.Time) error
}
