package createmonitor_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/St1lon/sentinel/internal/domain"
	createmonitor "github.com/St1lon/sentinel/internal/usecase/monitor/create"
)

type repoStub struct {
	created *domain.Monitor
	err     error
	calls   int
}

func (s *repoStub) Create(_ context.Context, monitor *domain.Monitor) error {
	s.calls++

	if s.err != nil {
		return s.err
	}

	s.created = monitor

	return nil
}

func ptr[T any](value T) *T {
	return &value
}

func validRequest() *createmonitor.Request {
	return &createmonitor.Request{
		UserID: "11111111-1111-1111-1111-111111111111",
		Name:   "  API продакшена  ",
		Target: " https://example.com/health ",
	}
}

func TestExecute_AppliesDefaults(t *testing.T) {
	t.Parallel()

	repo := &repoStub{}

	monitor, err := createmonitor.NewUsecase(repo).Execute(context.Background(), validRequest())
	require.NoError(t, err)

	require.Equal(t, "API продакшена", monitor.Name, "имя обрезается по краям")
	require.Equal(t, "https://example.com/health", monitor.Target)
	require.Equal(t, domain.MonitorKindHTTP, monitor.Kind)
	require.Equal(t, "GET", monitor.Method)
	require.Equal(t, domain.DefaultIntervalSeconds, monitor.IntervalSeconds)
	require.Equal(t, domain.DefaultTimeoutSeconds, monitor.TimeoutSeconds)
	require.Equal(t, domain.DefaultExpectedStatus, monitor.ExpectedStatus)
	require.Equal(t, domain.DefaultFailureThreshold, monitor.FailureThreshold)
	require.Equal(t, domain.MonitorStatusPending, monitor.Status)
	require.NotEmpty(t, monitor.ID)

	// Первая проверка должна уйти сразу, а не через interval.
	require.False(t, monitor.NextCheckAt.After(monitor.CreatedAt))
}

func TestExecute_PausedMonitorStartsPaused(t *testing.T) {
	t.Parallel()

	req := validRequest()
	req.Paused = true

	monitor, err := createmonitor.NewUsecase(&repoStub{}).Execute(context.Background(), req)
	require.NoError(t, err)
	require.Equal(t, domain.MonitorStatusPaused, monitor.Status)
}

func TestExecute_RejectsTimeoutNotBelowInterval(t *testing.T) {
	t.Parallel()

	repo := &repoStub{}
	req := validRequest()
	req.IntervalSeconds = ptr(30)
	req.TimeoutSeconds = ptr(30)

	_, err := createmonitor.NewUsecase(repo).Execute(context.Background(), req)

	require.ErrorIs(t, err, domain.ErrTimeoutExceedsInterval)
	require.Zero(t, repo.calls, "невалидный запрос не доходит до репозитория")
}

func TestExecute_RejectsOutOfRangeValues(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		mutate func(*createmonitor.Request)
		want   error
	}{
		"интервал меньше минимума": {
			func(r *createmonitor.Request) { r.IntervalSeconds = ptr(5) },
			domain.ErrInvalidInterval,
		},
		"интервал больше максимума": {
			func(r *createmonitor.Request) { r.IntervalSeconds = ptr(100000) },
			domain.ErrInvalidInterval,
		},
		"таймаут больше максимума": {
			func(r *createmonitor.Request) { r.TimeoutSeconds = ptr(300) },
			domain.ErrInvalidTimeout,
		},
		"ожидаемый код вне диапазона": {
			func(r *createmonitor.Request) { r.ExpectedStatus = ptr(999) },
			domain.ErrInvalidExpectedStatus,
		},
		"порог падений ноль": {
			func(r *createmonitor.Request) { r.FailureThreshold = ptr(0) },
			domain.ErrInvalidThreshold,
		},
		"порог падений слишком большой": {
			func(r *createmonitor.Request) { r.FailureThreshold = ptr(50) },
			domain.ErrInvalidThreshold,
		},
		"пустое имя": {
			func(r *createmonitor.Request) { r.Name = "   " },
			domain.ErrInvalidMonitorName,
		},
		"цель без схемы": {
			func(r *createmonitor.Request) { r.Target = "example.com" },
			domain.ErrInvalidTarget,
		},
		"нереализованный вид проверки": {
			func(r *createmonitor.Request) { r.Kind = "tls_cert" },
			domain.ErrInvalidMonitorKind,
		},
		"неизвестный вид проверки": {
			func(r *createmonitor.Request) { r.Kind = "smtp" },
			domain.ErrInvalidMonitorKind,
		},
		"небезопасный метод": {
			func(r *createmonitor.Request) { r.Method = "POST" },
			domain.ErrInvalidMethod,
		},
		"пустой пользователь": {
			func(r *createmonitor.Request) { r.UserID = "" },
			domain.ErrUnauthenticated,
		},
	}

	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			req := validRequest()
			testCase.mutate(req)

			_, err := createmonitor.NewUsecase(&repoStub{}).Execute(context.Background(), req)
			require.ErrorIs(t, err, testCase.want)
		})
	}
}

func TestExecute_PropagatesNameConflict(t *testing.T) {
	t.Parallel()

	repo := &repoStub{err: domain.ErrMonitorNameTaken}

	_, err := createmonitor.NewUsecase(repo).Execute(context.Background(), validRequest())
	require.ErrorIs(t, err, domain.ErrMonitorNameTaken)
}
