//go:build integration

package integrationtests

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/St1lon/sentinel/internal/domain"
	"github.com/St1lon/sentinel/internal/infra/db/postgres/repository"
	monitorsummary "github.com/St1lon/sentinel/internal/usecase/monitor/summary"
)

// Срез состояния считается по всей базе, а не по одному пользователю:
// это метрика системы. Поэтому проверяется приращение, а не абсолютные
// значения — в базе уже есть данные соседних тестов.
func TestFleetSummary_CountsStatusesAndOpenIncidents(t *testing.T) {
	ctx := context.Background()

	usecase := monitorsummary.NewUsecase(
		repository.NewMonitorRepo(testPool),
		repository.NewIncidentRepo(testPool),
	)

	before, err := usecase.Execute(ctx)
	require.NoError(t, err)

	user := newUser(ctx, t)

	down := newMonitor(ctx, t, user.ID, func(m *domain.Monitor) { m.Status = domain.MonitorStatusDown })
	newMonitor(ctx, t, user.ID, func(m *domain.Monitor) { m.Status = domain.MonitorStatusUp })
	newMonitor(ctx, t, user.ID, func(m *domain.Monitor) { m.Status = domain.MonitorStatusUp })
	newMonitor(ctx, t, user.ID, func(m *domain.Monitor) { m.Status = domain.MonitorStatusPaused; m.Paused = true })

	incident := &domain.Incident{
		ID:        uuid.NewString(),
		MonitorID: down.ID,
		StartedAt: time.Now().UTC(),
		Cause:     "connection refused",
	}
	require.NoError(t, repository.NewIncidentRepo(testPool).Open(ctx, incident))

	after, err := usecase.Execute(ctx)
	require.NoError(t, err)

	delta := func(status domain.MonitorStatus) int {
		return after.ByStatus[status] - before.ByStatus[status]
	}

	require.Equal(t, 1, delta(domain.MonitorStatusDown))
	require.Equal(t, 2, delta(domain.MonitorStatusUp))
	require.Equal(t, 1, delta(domain.MonitorStatusPaused))
	require.Equal(t, 1, after.OpenIncidents-before.OpenIncidents)

	// Закрытый инцидент из открытых уходит.
	require.NoError(t, repository.NewIncidentRepo(testPool).ResolveOpen(ctx, down.ID, time.Now().UTC()))

	resolved, err := usecase.Execute(ctx)
	require.NoError(t, err)
	require.Equal(t, before.OpenIncidents, resolved.OpenIncidents)
}
