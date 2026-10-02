//go:build integration

package integrationtests

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/St1lon/sentinel/internal/domain"
	"github.com/St1lon/sentinel/internal/infra/db/postgres"
	"github.com/St1lon/sentinel/internal/infra/db/postgres/repository"
	recordprobe "github.com/St1lon/sentinel/internal/usecase/probe/record"
)

func newRecordUsecase() *recordprobe.Usecase {
	return recordprobe.NewUsecase(
		repository.NewMonitorRepo(testPool),
		repository.NewCheckRepo(testPool),
		repository.NewIncidentRepo(testPool),
		postgres.NewTxManager(testPool),
	)
}

func failedCheck(monitorID string, at time.Time) *domain.Check {
	cause := "connection refused"

	return &domain.Check{MonitorID: monitorID, CheckedAt: at, Up: false, LatencyMS: 5000, Error: &cause}
}

func okCheck(monitorID string, at time.Time) *domain.Check {
	code := 200

	return &domain.Check{MonitorID: monitorID, CheckedAt: at, Up: true, StatusCode: &code, LatencyMS: 120}
}

func TestProbeFlow_IncidentLifecycle(t *testing.T) {
	ctx := context.Background()

	usecase := newRecordUsecase()
	monitors := repository.NewMonitorRepo(testPool)
	incidents := repository.NewIncidentRepo(testPool)

	user := newUser(ctx, t)
	monitor := newMonitor(ctx, t, user.ID, func(m *domain.Monitor) { m.FailureThreshold = 2 })
	now := time.Now().UTC()

	result, err := usecase.Execute(ctx, &recordprobe.Request{
		Monitor: monitor,
		Check:   failedCheck(monitor.ID, now),
	})
	require.NoError(t, err)
	require.False(t, result.Transition.OpenIncident)

	monitor, err = monitors.GetByIDForUser(ctx, monitor.ID, user.ID)
	require.NoError(t, err)
	require.Equal(t, 1, monitor.ConsecutiveFailures)
	require.Equal(t, domain.MonitorStatusPending, monitor.Status)

	open, err := incidents.GetOpenByMonitor(ctx, monitor.ID)
	require.NoError(t, err)
	require.Nil(t, open)

	result, err = usecase.Execute(ctx, &recordprobe.Request{
		Monitor: monitor,
		Check:   failedCheck(monitor.ID, now.Add(time.Minute)),
	})
	require.NoError(t, err)
	require.True(t, result.Transition.OpenIncident)

	monitor, err = monitors.GetByIDForUser(ctx, monitor.ID, user.ID)
	require.NoError(t, err)
	require.Equal(t, domain.MonitorStatusDown, monitor.Status)

	open, err = incidents.GetOpenByMonitor(ctx, monitor.ID)
	require.NoError(t, err)
	require.NotNil(t, open)
	require.Equal(t, "connection refused", open.Cause)
	require.True(t, open.IsOpen())

	_, err = usecase.Execute(ctx, &recordprobe.Request{
		Monitor: monitor,
		Check:   failedCheck(monitor.ID, now.Add(2*time.Minute)),
	})
	require.NoError(t, err)

	all, total, err := incidents.ListByMonitor(ctx, monitor.ID, 10, 0)
	require.NoError(t, err)
	require.Equal(t, 1, total, "частичный уникальный индекс не даёт открыть второй инцидент")
	require.Len(t, all, 1)

	monitor, err = monitors.GetByIDForUser(ctx, monitor.ID, user.ID)
	require.NoError(t, err)

	result, err = usecase.Execute(ctx, &recordprobe.Request{
		Monitor: monitor,
		Check:   okCheck(monitor.ID, now.Add(3*time.Minute)),
	})
	require.NoError(t, err)
	require.True(t, result.Transition.CloseIncident)

	monitor, err = monitors.GetByIDForUser(ctx, monitor.ID, user.ID)
	require.NoError(t, err)
	require.Equal(t, domain.MonitorStatusUp, monitor.Status)
	require.Zero(t, monitor.ConsecutiveFailures)

	open, err = incidents.GetOpenByMonitor(ctx, monitor.ID)
	require.NoError(t, err)
	require.Nil(t, open, "после восстановления открытых инцидентов быть не должно")

	closed, _, err := incidents.ListByMonitor(ctx, monitor.ID, 10, 0)
	require.NoError(t, err)
	require.NotNil(t, closed[0].ResolvedAt)
	require.Positive(t, closed[0].Duration(time.Now().UTC()))
}

func TestProbeFlow_TransactionRollsBackOnFailure(t *testing.T) {
	ctx := context.Background()

	user := newUser(ctx, t)
	monitor := newMonitor(ctx, t, user.ID)

	require.NoError(t, repository.NewMonitorRepo(testPool).Delete(ctx, monitor.ID, user.ID))

	_, err := newRecordUsecase().Execute(ctx, &recordprobe.Request{
		Monitor: monitor,
		Check:   okCheck(monitor.ID, time.Now().UTC()),
	})

	require.ErrorIs(t, err, domain.ErrMonitorNotFound)

	checks, err := repository.NewCheckRepo(testPool).ListByMonitor(
		ctx, monitor.ID, time.Now().UTC().Add(-time.Hour), time.Now().UTC().Add(time.Hour), 10,
	)
	require.NoError(t, err)
	require.Empty(t, checks, "в откатившейся транзакции не должно остаться записей")
}

func TestCheckRepo_StatsAndBuckets(t *testing.T) {
	ctx := context.Background()
	checks := repository.NewCheckRepo(testPool)

	user := newUser(ctx, t)
	monitor := newMonitor(ctx, t, user.ID)

	base := time.Now().UTC().Truncate(time.Hour).Add(-3 * time.Hour)

	insertCheck(ctx, t, monitor.ID, base.Add(5*time.Minute), true, 100)
	insertCheck(ctx, t, monitor.ID, base.Add(10*time.Minute), false, 5000)
	insertCheck(ctx, t, monitor.ID, base.Add(time.Hour+5*time.Minute), true, 150)
	insertCheck(ctx, t, monitor.ID, base.Add(2*time.Hour+5*time.Minute), true, 200)

	from := base.Add(-time.Minute)
	to := time.Now().UTC().Add(time.Minute)

	stats, err := checks.Stats(ctx, monitor.ID, from, to)
	require.NoError(t, err)
	require.Equal(t, 4, stats.TotalChecks)
	require.Equal(t, 3, stats.SuccessfulChecks)
	require.InDelta(t, 0.75, stats.UptimeRatio, 1e-9)
	require.InDelta(t, 1362.5, stats.AvgLatencyMS, 0.1)
	require.Equal(t, 5000, stats.P95LatencyMS)
	require.NotNil(t, stats.LastCheckedAt)

	buckets, err := checks.Buckets(ctx, monitor.ID, from, to, domain.BucketSizeHour)
	require.NoError(t, err)
	require.Len(t, buckets, 3)
	require.Equal(t, 2, buckets[0].Total)
	require.Equal(t, 1, buckets[0].Successful)
	require.InDelta(t, 0.5, buckets[0].UptimeRatio(), 1e-9)

	multi, err := checks.BucketsForMonitors(ctx, []string{monitor.ID}, from, to, domain.BucketSizeDay)
	require.NoError(t, err)
	require.NotEmpty(t, multi)
	require.Equal(t, monitor.ID, multi[0].MonitorID)

	_, err = checks.Buckets(ctx, monitor.ID, from, to, domain.BucketSize("day'; DROP TABLE checks; --"))
	require.ErrorIs(t, err, domain.ErrInvalidBucket)
}

func TestCheckRepo_DeleteOlderThan(t *testing.T) {
	ctx := context.Background()
	checks := repository.NewCheckRepo(testPool)

	user := newUser(ctx, t)
	monitor := newMonitor(ctx, t, user.ID)

	old := time.Now().UTC().Add(-48 * time.Hour)
	recent := time.Now().UTC().Add(-time.Hour)

	insertCheck(ctx, t, monitor.ID, old, true, 100)
	insertCheck(ctx, t, monitor.ID, old.Add(time.Minute), true, 110)
	insertCheck(ctx, t, monitor.ID, recent, true, 120)

	deleted, err := checks.DeleteOlderThan(ctx, time.Now().UTC().Add(-24*time.Hour), 1000)
	require.NoError(t, err)
	require.GreaterOrEqual(t, deleted, int64(2))

	remaining, err := checks.ListByMonitor(ctx, monitor.ID, time.Now().UTC().Add(-72*time.Hour), time.Now().UTC(), 100)
	require.NoError(t, err)
	require.Len(t, remaining, 1, "свежая проверка должна остаться")
}

func TestUserRepo_UniqueEmailAndSlugLookup(t *testing.T) {
	ctx := context.Background()
	users := repository.NewUserRepo(testPool)

	user := newUser(ctx, t)

	byEmail, err := users.GetByEmail(ctx, user.Email)
	require.NoError(t, err)
	require.Equal(t, user.ID, byEmail.ID)

	bySlug, err := users.GetByStatusPageSlug(ctx, user.StatusPageSlug)
	require.NoError(t, err)
	require.Equal(t, user.ID, bySlug.ID)

	_, err = users.GetByStatusPageSlug(ctx, "no-such-slug")
	require.ErrorIs(t, err, domain.ErrStatusPageNotFound)

	duplicate := *user
	duplicate.ID = uuid.NewString()
	duplicate.StatusPageSlug = uuid.NewString()
	require.ErrorIs(t, users.Create(ctx, &duplicate), domain.ErrEmailAlreadyUsed)
}
