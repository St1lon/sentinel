package monitorsummary_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/St1lon/sentinel/internal/domain"
	monitorsummary "github.com/St1lon/sentinel/internal/usecase/monitor/summary"
)

type monitorRepoStub struct {
	counts map[domain.MonitorStatus]int
	err    error
}

func (s *monitorRepoStub) CountByStatus(context.Context) (map[domain.MonitorStatus]int, error) {
	return s.counts, s.err
}

type incidentRepoStub struct {
	open  int
	err   error
	calls int
}

func (s *incidentRepoStub) CountOpen(context.Context) (int, error) {
	s.calls++

	return s.open, s.err
}

func TestExecute_CollectsCountsFromBothRepositories(t *testing.T) {
	t.Parallel()

	monitors := &monitorRepoStub{counts: map[domain.MonitorStatus]int{
		domain.MonitorStatusUp:   4,
		domain.MonitorStatusDown: 1,
	}}

	summary, err := monitorsummary.NewUsecase(monitors, &incidentRepoStub{open: 1}).Execute(context.Background())

	require.NoError(t, err)
	require.Equal(t, 4, summary.ByStatus[domain.MonitorStatusUp])
	require.Equal(t, 1, summary.ByStatus[domain.MonitorStatusDown])
	require.Zero(t, summary.ByStatus[domain.MonitorStatusPaused], "статусов без мониторов в срезе нет")
	require.Equal(t, 1, summary.OpenIncidents)
}

func TestExecute_StopsOnMonitorsError(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("db is down")
	incidents := &incidentRepoStub{}

	_, err := monitorsummary.NewUsecase(&monitorRepoStub{err: wantErr}, incidents).Execute(context.Background())

	require.ErrorIs(t, err, wantErr)
	require.Zero(t, incidents.calls, "второй запрос не идёт, если первый не удался")
}
