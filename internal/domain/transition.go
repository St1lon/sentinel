package domain

import (
	"fmt"
	"strings"
)

// Transition — решение о смене состояния монитора после одной проверки.
// Чистая структура данных: воркер применяет её одной транзакцией.
type Transition struct {
	Status              MonitorStatus
	ConsecutiveFailures int
	StatusChanged       bool
	OpenIncident        bool
	CloseIncident       bool
	Cause               string
}

// EvaluateProbe — ядро доменной логики: по текущему состоянию монитора и результату
// проверки решает, меняется ли статус и нужно ли открыть либо закрыть инцидент.
//
// Правила:
//   - успешная проверка сбрасывает счётчик падений; если монитор был down —
//     инцидент закрывается, статус становится up;
//   - неуспешная увеличивает счётчик; инцидент открывается только когда счётчик
//     достиг FailureThreshold (защита от одиночных сетевых сбоев — флаппинга);
//   - повторная ошибка при уже открытом инциденте ничего не открывает заново.
//
// Функция не мутирует монитор: вызывающий применяет Transition сам.
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

	// Монитор ещё не признан упавшим, но порог достигнут — открываем инцидент.
	if failures >= monitor.FailureThreshold && monitor.Status != MonitorStatusDown {
		transition.Status = MonitorStatusDown
		transition.StatusChanged = true
		transition.OpenIncident = true
		transition.Cause = describeFailure(monitor, check)

		return transition
	}

	// Порог ещё не достигнут: монитор остаётся в прежнем статусе.
	// Из pending до первого успеха или до порога не выходим.
	return transition
}

// describeFailure формирует человекочитаемую причину инцидента для статус-страницы.
func describeFailure(monitor *Monitor, check *Check) string {
	if check.Error != nil && strings.TrimSpace(*check.Error) != "" {
		return *check.Error
	}

	if check.StatusCode != nil {
		return fmt.Sprintf("unexpected status %d, expected %d", *check.StatusCode, monitor.ExpectedStatus)
	}

	return "check failed"
}
