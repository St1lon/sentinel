package createmonitor

import (
	"github.com/St1lon/sentinel/internal/domain"
	"github.com/St1lon/sentinel/internal/usecase/common/validate"
)

// Request — входные данные создания монитора.
// Необязательные числовые поля — указатели, чтобы отличить «не передано»
// (подставляем значение по умолчанию) от переданного нуля (ошибка валидации).
type Request struct {
	UserID           string
	Name             string
	Kind             string
	Target           string
	Method           string
	IntervalSeconds  *int
	TimeoutSeconds   *int
	ExpectedStatus   *int
	FailureThreshold *int
	IsPublic         bool
	Paused           bool

	// Нормализованные значения, заполняются validate().
	kind             domain.MonitorKind
	method           string
	intervalSeconds  int
	timeoutSeconds   int
	expectedStatus   int
	failureThreshold int
}

// validate проверяет и нормализует входные данные, подставляя значения по умолчанию.
func (req *Request) validate() error {
	if !validate.UUID(req.UserID) {
		return domain.ErrUnauthenticated
	}

	name, err := validate.MonitorName(req.Name)
	if err != nil {
		return err
	}

	req.Name = name

	target, err := validate.MonitorTarget(req.Target)
	if err != nil {
		return err
	}

	req.Target = target

	req.kind, err = validate.MonitorKind(orDefault(req.Kind, string(domain.MonitorKindHTTP)))
	if err != nil {
		return err
	}

	req.method, err = validate.Method(orDefault(req.Method, domain.DefaultMethod))
	if err != nil {
		return err
	}

	req.intervalSeconds = intOrDefault(req.IntervalSeconds, domain.DefaultIntervalSeconds)
	req.timeoutSeconds = intOrDefault(req.TimeoutSeconds, domain.DefaultTimeoutSeconds)
	req.expectedStatus = intOrDefault(req.ExpectedStatus, domain.DefaultExpectedStatus)
	req.failureThreshold = intOrDefault(req.FailureThreshold, domain.DefaultFailureThreshold)

	if err := validate.Schedule(req.intervalSeconds, req.timeoutSeconds); err != nil {
		return err
	}

	if err := validate.ExpectedStatus(req.expectedStatus); err != nil {
		return err
	}

	return validate.FailureThreshold(req.failureThreshold)
}

func orDefault(value, fallback string) string {
	if value == "" {
		return fallback
	}

	return value
}

func intOrDefault(value *int, fallback int) int {
	if value == nil {
		return fallback
	}

	return *value
}
