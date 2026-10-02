package recordprobe_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/St1lon/sentinel/internal/domain"
	recordprobe "github.com/St1lon/sentinel/internal/usecase/probe/record"
)

type monitorRepoStub struct {
	status   domain.MonitorStatus
	failures int
	calls    int
	err      error
}

func (s *monitorRepoStub) ApplyProbeResult(
	_ context.Context, _ string, status domain.MonitorStatus, failures int, _ time.Time,
) error {
	s.calls++
	s.status = status
	s.failures = failures

	return s.err
}

type checkRepoStub struct {
	inserted *domain.Check
	err      error
}

func (s *checkRepoStub) Insert(_ context.Context, check *domain.Check) error {
	if s.err != nil {
		return s.err
	}

	check.ID = 42
	s.inserted = check

	return nil
}

type incidentRepoStub struct {
	opened   *domain.Incident
	resolved int
	err      error
}

func (s *incidentRepoStub) Open(_ context.Context, incident *domain.Incident) error {
	if s.err != nil {
		return s.err
	}

	s.opened = incident

	return nil
}

func (s *incidentRepoStub) ResolveOpen(_ context.Context, _ string, _ time.Time) error {
	if s.err != nil {
		return s.err
	}

	s.resolved++

	return nil
}

// txStub просто вызывает функцию: транзакционность проверяется
// интеграционными тестами на реальной БД, здесь важна последовательность шагов.
type txStub struct {
	calls int
}

func (s *txStub) WithTx(ctx context.Context, fn func(context.Context) error) error {
	s.calls++

	return fn(ctx)
}

func monitorAt(status domain.MonitorStatus, failures, threshold int) *domain.Monitor {
	return &domain.Monitor{
		ID:                  "11111111-1111-1111-1111-111111111111",
		Status:              status,
		ConsecutiveFailures: failures,
		FailureThreshold:    threshold,
		ExpectedStatus:      200,
	}
}

func checkAt(up bool) *domain.Check {
	return &domain.Check{Up: up, CheckedAt: time.Now().UTC(), LatencyMS: 12}
}

func TestExecute_SuccessfulCheckUpdatesStatusInOneTransaction(t *testing.T) {
	t.Parallel()

	monitors := &monitorRepoStub{}
	checks := &checkRepoStub{}
	incidents := &incidentRepoStub{}
	tx := &txStub{}

	result, err := recordprobe.NewUsecase(monitors, checks, incidents, tx).Execute(
		context.Background(),
		&recordprobe.Request{Monitor: monitorAt(domain.MonitorStatusPending, 0, 2), Check: checkAt(true)},
	)

	require.NoError(t, err)
	require.Equal(t, 1, tx.calls, "все записи идут одной транзакцией")
	require.Equal(t, domain.MonitorStatusUp, monitors.status)
	require.Zero(t, monitors.failures)
	require.NotNil(t, checks.inserted)
	require.Equal(t, int64(42), result.CheckID)
	require.Nil(t, incidents.opened)
	require.Zero(t, incidents.resolved, "инцидента не было — закрывать нечего")
}

func TestExecute_ThresholdReachedOpensIncident(t *testing.T) {
	t.Parallel()

	monitors := &monitorRepoStub{}
	incidents := &incidentRepoStub{}

	result, err := recordprobe.NewUsecase(monitors, &checkRepoStub{}, incidents, &txStub{}).Execute(
		context.Background(),
		&recordprobe.Request{Monitor: monitorAt(domain.MonitorStatusUp, 1, 2), Check: checkAt(false)},
	)

	require.NoError(t, err)
	require.True(t, result.Transition.OpenIncident)
	require.Equal(t, domain.MonitorStatusDown, monitors.status)
	require.Equal(t, 2, monitors.failures)
	require.NotNil(t, incidents.opened)
	require.NotEmpty(t, incidents.opened.ID)
	require.Equal(t, "check failed", incidents.opened.Cause)
}

func TestExecute_FailureBelowThresholdOnlyCountsUp(t *testing.T) {
	t.Parallel()

	monitors := &monitorRepoStub{}
	incidents := &incidentRepoStub{}

	_, err := recordprobe.NewUsecase(monitors, &checkRepoStub{}, incidents, &txStub{}).Execute(
		context.Background(),
		&recordprobe.Request{Monitor: monitorAt(domain.MonitorStatusUp, 0, 3), Check: checkAt(false)},
	)

	require.NoError(t, err)
	require.Equal(t, domain.MonitorStatusUp, monitors.status)
	require.Equal(t, 1, monitors.failures)
	require.Nil(t, incidents.opened, "одиночный сбой не открывает инцидент")
}

func TestExecute_RecoveryResolvesIncident(t *testing.T) {
	t.Parallel()

	monitors := &monitorRepoStub{}
	incidents := &incidentRepoStub{}

	_, err := recordprobe.NewUsecase(monitors, &checkRepoStub{}, incidents, &txStub{}).Execute(
		context.Background(),
		&recordprobe.Request{Monitor: monitorAt(domain.MonitorStatusDown, 3, 2), Check: checkAt(true)},
	)

	require.NoError(t, err)
	require.Equal(t, domain.MonitorStatusUp, monitors.status)
	require.Equal(t, 1, incidents.resolved)
}

func TestExecute_CheckInsertFailureAbortsEverything(t *testing.T) {
	t.Parallel()

	monitors := &monitorRepoStub{}

	_, err := recordprobe.NewUsecase(
		monitors,
		&checkRepoStub{err: errors.New("db is down")},
		&incidentRepoStub{},
		&txStub{},
	).Execute(
		context.Background(),
		&recordprobe.Request{Monitor: monitorAt(domain.MonitorStatusUp, 0, 2), Check: checkAt(true)},
	)

	require.Error(t, err)
	require.Zero(t, monitors.calls, "статус не меняется, если проверка не записалась")
}

func TestExecute_ValidatesRequest(t *testing.T) {
	t.Parallel()

	usecase := recordprobe.NewUsecase(&monitorRepoStub{}, &checkRepoStub{}, &incidentRepoStub{}, &txStub{})

	cases := map[string]*recordprobe.Request{
		"без монитора":             {Check: checkAt(true)},
		"без проверки":             {Monitor: monitorAt(domain.MonitorStatusUp, 0, 2)},
		"проверка чужого монитора": {Monitor: monitorAt(domain.MonitorStatusUp, 0, 2), Check: &domain.Check{MonitorID: "other", CheckedAt: time.Now()}},
		"проверка без времени":     {Monitor: monitorAt(domain.MonitorStatusUp, 0, 2), Check: &domain.Check{}},
	}

	for name, req := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			_, err := usecase.Execute(context.Background(), req)
			require.Error(t, err)
		})
	}
}
