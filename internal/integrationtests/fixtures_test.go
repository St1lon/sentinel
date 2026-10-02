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
)

// Каждый тест сам создаёт нужные ему данные с уникальными идентификаторами:
// общих фикстур между тестами нет, порядок запуска не важен.

func newUser(ctx context.Context, t *testing.T) *domain.User {
	t.Helper()

	user := &domain.User{
		ID:             uuid.NewString(),
		Email:          uuid.NewString() + "@example.com",
		PasswordHash:   "hash",
		StatusPageSlug: uuid.NewString(),
		CreatedAt:      time.Now().UTC().Truncate(time.Microsecond),
	}

	require.NoError(t, repository.NewUserRepo(testPool).Create(ctx, user))

	return user
}

func newMonitor(ctx context.Context, t *testing.T, userID string, mutate ...func(*domain.Monitor)) *domain.Monitor {
	t.Helper()

	now := time.Now().UTC().Truncate(time.Microsecond)

	monitor := &domain.Monitor{
		ID:               uuid.NewString(),
		UserID:           userID,
		Name:             "monitor-" + uuid.NewString(),
		Kind:             domain.MonitorKindHTTP,
		Target:           "https://example.com/health",
		Method:           "GET",
		IntervalSeconds:  60,
		TimeoutSeconds:   10,
		ExpectedStatus:   200,
		FailureThreshold: 2,
		Status:           domain.MonitorStatusPending,
		NextCheckAt:      now,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	for _, fn := range mutate {
		fn(monitor)
	}

	require.NoError(t, repository.NewMonitorRepo(testPool).Create(ctx, monitor))

	return monitor
}

func insertCheck(ctx context.Context, t *testing.T, monitorID string, at time.Time, up bool, latency int) {
	t.Helper()

	statusCode := 200
	if !up {
		statusCode = 503
	}

	check := &domain.Check{
		MonitorID:  monitorID,
		CheckedAt:  at,
		Up:         up,
		StatusCode: &statusCode,
		LatencyMS:  latency,
	}

	require.NoError(t, repository.NewCheckRepo(testPool).Insert(ctx, check))
	require.NotZero(t, check.ID, "Insert должен вернуть присвоенный id")
}
