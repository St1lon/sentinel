package domain

import (
	"fmt"
	"strings"
)

type Transition struct {
	Status              MonitorStatus
	ConsecutiveFailures int
	StatusChanged       bool
	OpenIncident        bool
	CloseIncident       bool
	Cause               string
}

// Инцидент открывается только после FailureThreshold подряд неудач — защита от флаппинга.
func EvaluateProbe(monitor *Monitor, check *Check) Transition {
	if check.Up {
		return successTransition(monitor)
	}

	return failureTransition(monitor, check)
}

func successTransition(monitor *Monitor) Transition {
	transition := Transition{
		Status:              MonitorStatusUp,
		ConsecutiveFailures: 0,
		StatusChanged:       monitor.Status != MonitorStatusUp,
		CloseIncident:       monitor.Status == MonitorStatusDown,
	}

	return transition
}

func failureTransition(monitor *Monitor, check *Check) Transition {
	failures := monitor.ConsecutiveFailures + 1

	transition := Transition{
		Status:              monitor.Status,
		ConsecutiveFailures: failures,
	}

	if failures >= monitor.FailureThreshold && monitor.Status != MonitorStatusDown {
		transition.Status = MonitorStatusDown
		transition.StatusChanged = true
		transition.OpenIncident = true
		transition.Cause = describeFailure(monitor, check)

		return transition
	}

	return transition
}

func describeFailure(monitor *Monitor, check *Check) string {
	if check.Error != nil && strings.TrimSpace(*check.Error) != "" {
		return *check.Error
	}

	if check.StatusCode != nil {
		return fmt.Sprintf("unexpected status %d, expected %d", *check.StatusCode, monitor.ExpectedStatus)
	}

	return "check failed"
}
