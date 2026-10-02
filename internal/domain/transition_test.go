package domain_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/St1lon/sentinel/internal/domain"
)

func monitorWith(status domain.MonitorStatus, failures, threshold int) *domain.Monitor {
	return &domain.Monitor{
		ID:                  "11111111-1111-1111-1111-111111111111",
		Status:              status,
		ConsecutiveFailures: failures,
		FailureThreshold:    threshold,
		ExpectedStatus:      200,
	}
}

func failedCheck(statusCode *int, errText *string) *domain.Check {
	return &domain.Check{Up: false, StatusCode: statusCode, Error: errText}
}

func TestEvaluateProbe_FirstSuccessFromPending(t *testing.T) {
	t.Parallel()

	transition := domain.EvaluateProbe(
		monitorWith(domain.MonitorStatusPending, 0, 2),
		&domain.Check{Up: true},
	)

	require.Equal(t, domain.MonitorStatusUp, transition.Status)
	require.True(t, transition.StatusChanged)
	require.False(t, transition.OpenIncident)
	require.False(t, transition.CloseIncident, "из pending нечего закрывать")
	require.Zero(t, transition.ConsecutiveFailures)
}

func TestEvaluateProbe_SuccessOnAlreadyUpChangesNothing(t *testing.T) {
	t.Parallel()

	transition := domain.EvaluateProbe(
		monitorWith(domain.MonitorStatusUp, 0, 2),
		&domain.Check{Up: true},
	)

	require.Equal(t, domain.MonitorStatusUp, transition.Status)
	require.False(t, transition.StatusChanged)
	require.False(t, transition.OpenIncident)
	require.False(t, transition.CloseIncident)
}

func TestEvaluateProbe_RecoveryClosesIncident(t *testing.T) {
	t.Parallel()

	transition := domain.EvaluateProbe(
		monitorWith(domain.MonitorStatusDown, 5, 2),
		&domain.Check{Up: true},
	)

	require.Equal(t, domain.MonitorStatusUp, transition.Status)
	require.True(t, transition.StatusChanged)
	require.True(t, transition.CloseIncident)
	require.Zero(t, transition.ConsecutiveFailures)
}

func TestEvaluateProbe_SingleFailureBelowThresholdDoesNotOpenIncident(t *testing.T) {
	t.Parallel()

	transition := domain.EvaluateProbe(
		monitorWith(domain.MonitorStatusUp, 0, 3),
		failedCheck(nil, nil),
	)

	require.Equal(t, domain.MonitorStatusUp, transition.Status, "флаппинг не должен ронять монитор")
	require.False(t, transition.StatusChanged)
	require.False(t, transition.OpenIncident)
	require.Equal(t, 1, transition.ConsecutiveFailures)
}

func TestEvaluateProbe_ThresholdReachedOpensIncident(t *testing.T) {
	t.Parallel()

	statusCode := 503

	transition := domain.EvaluateProbe(
		monitorWith(domain.MonitorStatusUp, 1, 2),
		failedCheck(&statusCode, nil),
	)

	require.Equal(t, domain.MonitorStatusDown, transition.Status)
	require.True(t, transition.StatusChanged)
	require.True(t, transition.OpenIncident)
	require.Equal(t, 2, transition.ConsecutiveFailures)
	require.Equal(t, "unexpected status 503, expected 200", transition.Cause)
}

func TestEvaluateProbe_FailureWhileDownDoesNotReopenIncident(t *testing.T) {
	t.Parallel()

	transition := domain.EvaluateProbe(
		monitorWith(domain.MonitorStatusDown, 4, 2),
		failedCheck(nil, nil),
	)

	require.Equal(t, domain.MonitorStatusDown, transition.Status)
	require.False(t, transition.StatusChanged)
	require.False(t, transition.OpenIncident, "инцидент уже открыт")
	require.Equal(t, 5, transition.ConsecutiveFailures)
}

func TestEvaluateProbe_CausePrefersNetworkError(t *testing.T) {
	t.Parallel()

	errText := "dial tcp: i/o timeout"
	statusCode := 500

	transition := domain.EvaluateProbe(
		monitorWith(domain.MonitorStatusUp, 0, 1),
		failedCheck(&statusCode, &errText),
	)

	require.True(t, transition.OpenIncident)
	require.Equal(t, errText, transition.Cause)
}

func TestEvaluateProbe_ThresholdOneOpensImmediately(t *testing.T) {
	t.Parallel()

	transition := domain.EvaluateProbe(
		monitorWith(domain.MonitorStatusPending, 0, 1),
		failedCheck(nil, nil),
	)

	require.Equal(t, domain.MonitorStatusDown, transition.Status)
	require.True(t, transition.OpenIncident)
	require.Equal(t, "check failed", transition.Cause)
}

func TestBucketUptimeRatio(t *testing.T) {
	t.Parallel()

	require.InDelta(t, 0.0, (&domain.Bucket{}).UptimeRatio(), 1e-9)
	require.InDelta(t, 0.5, (&domain.Bucket{Total: 10, Successful: 5}).UptimeRatio(), 1e-9)
}

func TestBucketSizeValid(t *testing.T) {
	t.Parallel()

	require.True(t, domain.BucketSizeHour.Valid())
	require.True(t, domain.BucketSizeDay.Valid())
	require.False(t, domain.BucketSize("month").Valid())
	require.False(t, domain.BucketSize("day; DROP TABLE checks").Valid())
}
