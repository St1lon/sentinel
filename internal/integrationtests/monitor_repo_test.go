//go:build integration

package integrationtests

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/St1lon/sentinel/internal/domain"
	"github.com/St1lon/sentinel/internal/infra/db/postgres/repository"
)

func TestMonitorRepo_CRUD(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewMonitorRepo(testPool)

	user := newUser(ctx, t)
	monitor := newMonitor(ctx, t, user.ID)

	loaded, err := repo.GetByIDForUser(ctx, monitor.ID, user.ID)
	require.NoError(t, err)
	require.Equal(t, monitor.Name, loaded.Name)
	require.Equal(t, domain.MonitorKindHTTP, loaded.Kind)
	require.Equal(t, domain.MonitorStatusPending, loaded.Status)

	loaded.Name += "-updated"
	loaded.IntervalSeconds = 120
	loaded.IsPublic = true
	loaded.UpdatedAt = time.Now().UTC()

	require.NoError(t, repo.Update(ctx, loaded))

	reloaded, err := repo.GetByIDForUser(ctx, monitor.ID, user.ID)
	require.NoError(t, err)
	require.Equal(t, loaded.Name, reloaded.Name)
	require.Equal(t, 120, reloaded.IntervalSeconds)
	require.True(t, reloaded.IsPublic)

	monitors, total, err := repo.ListByUser(ctx, user.ID, 10, 0)
	require.NoError(t, err)
	require.Equal(t, 1, total)
	require.Len(t, monitors, 1)

	require.NoError(t, repo.Delete(ctx, monitor.ID, user.ID))

	_, err = repo.GetByIDForUser(ctx, monitor.ID, user.ID)
	require.ErrorIs(t, err, domain.ErrMonitorNotFound)

	require.ErrorIs(t, repo.Delete(ctx, monitor.ID, user.ID), domain.ErrMonitorNotFound,
		"повторное удаление должно давать not found, а не тихо проходить")
}

func TestMonitorRepo_ForeignUserCannotSeeMonitor(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewMonitorRepo(testPool)

	owner := newUser(ctx, t)
	stranger := newUser(ctx, t)
	monitor := newMonitor(ctx, t, owner.ID)

	_, err := repo.GetByIDForUser(ctx, monitor.ID, stranger.ID)
	require.ErrorIs(t, err, domain.ErrMonitorNotFound)

	require.ErrorIs(t, repo.Delete(ctx, monitor.ID, stranger.ID), domain.ErrMonitorNotFound)
}

func TestMonitorRepo_DuplicateNamePerUserRejected(t *testing.T) {
	ctx := context.Background()

	user := newUser(ctx, t)
	first := newMonitor(ctx, t, user.ID)

	err := repository.NewMonitorRepo(testPool).Create(ctx, &domain.Monitor{
		ID:               uuid.NewString(),
		UserID:           user.ID,
		Name:             first.Name,
		Kind:             domain.MonitorKindHTTP,
		Target:           "https://other.example.com",
		Method:           "GET",
		IntervalSeconds:  60,
		TimeoutSeconds:   10,
		ExpectedStatus:   200,
		FailureThreshold: 2,
		Status:           domain.MonitorStatusPending,
		NextCheckAt:      time.Now().UTC(),
		CreatedAt:        time.Now().UTC(),
		UpdatedAt:        time.Now().UTC(),
	})

	require.ErrorIs(t, err, domain.ErrMonitorNameTaken)
}

func TestMonitorRepo_CheckConstraintsEnforcedByDatabase(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewMonitorRepo(testPool)
	user := newUser(ctx, t)

	broken := &domain.Monitor{
		ID:               uuid.NewString(),
		UserID:           user.ID,
		Name:             "broken",
		Kind:             domain.MonitorKindHTTP,
		Target:           "https://example.com",
		Method:           "GET",
		IntervalSeconds:  10,
		TimeoutSeconds:   30, // таймаут больше интервала — запрещено CHECK'ом
		ExpectedStatus:   200,
		FailureThreshold: 2,
		Status:           domain.MonitorStatusPending,
		NextCheckAt:      time.Now().UTC(),
		CreatedAt:        time.Now().UTC(),
		UpdatedAt:        time.Now().UTC(),
	}

	require.Error(t, repo.Create(ctx, broken))
}

func TestMonitorRepo_LeaseDueGivesEachMonitorToSingleWorker(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewMonitorRepo(testPool)
	user := newUser(ctx, t)

	past := time.Now().UTC().Add(-time.Hour)
	const monitorCount = 6

	expected := make(map[string]struct{}, monitorCount)

	for range monitorCount {
		monitor := newMonitor(ctx, t, user.ID, func(m *domain.Monitor) { m.NextCheckAt = past })
		expected[monitor.ID] = struct{}{}
	}

	const workers = 3

	var (
		mu    sync.Mutex
		taken []string
		wg    sync.WaitGroup
	)

	wg.Add(workers)

	for range workers {
		go func() {
			defer wg.Done()

			leased, err := repo.LeaseDue(ctx, time.Now().UTC(), monitorCount)
			if err != nil {
				return
			}

			mu.Lock()
			defer mu.Unlock()

			for _, monitor := range leased {
				if _, ours := expected[monitor.ID]; ours {
					taken = append(taken, monitor.ID)
				}
			}
		}()
	}

	wg.Wait()

	unique := make(map[string]struct{}, len(taken))
	for _, id := range taken {
		_, duplicate := unique[id]
		require.False(t, duplicate, "монитор %s выдан дважды", id)

		unique[id] = struct{}{}
	}

	require.Len(t, unique, monitorCount, "все просроченные мониторы должны быть разобраны")

	again, err := repo.LeaseDue(ctx, time.Now().UTC(), monitorCount)
	require.NoError(t, err)

	for _, monitor := range again {
		_, ours := expected[monitor.ID]
		require.False(t, ours, "монитор %s выдан повторно до наступления интервала", monitor.ID)
	}
}

func TestMonitorRepo_LeaseDueSkipsPaused(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewMonitorRepo(testPool)
	user := newUser(ctx, t)

	past := time.Now().UTC().Add(-time.Hour)

	paused := newMonitor(ctx, t, user.ID, func(m *domain.Monitor) {
		m.NextCheckAt = past
		m.Paused = true
		m.Status = domain.MonitorStatusPaused
	})

	leased, err := repo.LeaseDue(ctx, time.Now().UTC(), 100)
	require.NoError(t, err)

	for _, monitor := range leased {
		require.NotEqual(t, paused.ID, monitor.ID, "монитор на паузе проверять нельзя")
	}
}

func TestMonitorRepo_ApplyProbeResult(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewMonitorRepo(testPool)

	user := newUser(ctx, t)
	monitor := newMonitor(ctx, t, user.ID)

	checkedAt := time.Now().UTC().Truncate(time.Millisecond)

	require.NoError(t, repo.ApplyProbeResult(ctx, monitor.ID, domain.MonitorStatusDown, 3, checkedAt))

	updated, err := repo.GetByIDForUser(ctx, monitor.ID, user.ID)
	require.NoError(t, err)
	require.Equal(t, domain.MonitorStatusDown, updated.Status)
	require.Equal(t, 3, updated.ConsecutiveFailures)
	require.NotNil(t, updated.LastCheckedAt)
	require.WithinDuration(t, checkedAt, *updated.LastCheckedAt, time.Second)
}

func TestMonitorRepo_ListPublicByUser(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewMonitorRepo(testPool)

	user := newUser(ctx, t)
	public := newMonitor(ctx, t, user.ID, func(m *domain.Monitor) { m.IsPublic = true })
	newMonitor(ctx, t, user.ID)

	monitors, err := repo.ListPublicByUser(ctx, user.ID)
	require.NoError(t, err)
	require.Len(t, monitors, 1)
	require.Equal(t, public.ID, monitors[0].ID)
}
