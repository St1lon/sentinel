// Package validate содержит правила валидации, общие для нескольких usecase.
// Вынесено сюда, а не продублировано в create/update, чтобы правило менялось
// в одном месте; сами Request'ы вызывают эти функции из своих validate().
package validate

import (
	"net/mail"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/St1lon/sentinel/internal/domain"
)

const (
	// MinPasswordLength — минимальная длина пароля.
	MinPasswordLength = 8
	// MaxPasswordLength — ограничение bcrypt: байты свыше 72 игнорируются,
	// поэтому длинный пароль лучше отклонить явно, чем молча обрезать.
	MaxPasswordLength = 72
	// MaxEmailLength — практический предел длины адреса.
	MaxEmailLength = 254

	// MinStatusCode и MaxStatusCode — границы ожидаемого HTTP-кода.
	MinStatusCode = 100
	MaxStatusCode = 599

	// MaxPageLimit — максимальный размер страницы в списках.
	MaxPageLimit = 200
	// DefaultPageLimit — размер страницы по умолчанию.
	DefaultPageLimit = 50
	// MaxTimeRange — максимальная глубина выборки статистики.
	MaxTimeRange = 400 * 24 * time.Hour
)

// allowedMethods — методы, допустимые для HTTP-проверки.
// Проверка должна быть безопасной и идемпотентной, поэтому ни POST, ни DELETE.
var allowedMethods = []string{"GET", "HEAD", "OPTIONS"}

// Email проверяет и нормализует адрес.
func Email(raw string) (string, error) {
	email := strings.ToLower(strings.TrimSpace(raw))

	if email == "" || len(email) > MaxEmailLength {
		return "", domain.ErrInvalidEmail
	}

	addr, err := mail.ParseAddress(email)
	if err != nil || addr.Address != email {
		return "", domain.ErrInvalidEmail
	}

	if !strings.Contains(email, ".") {
		return "", domain.ErrInvalidEmail
	}

	return email, nil
}

// Password проверяет пароль на минимальные требования.
func Password(password string) error {
	if len(password) < MinPasswordLength || len(password) > MaxPasswordLength {
		return domain.ErrWeakPassword
	}

	return nil
}

// MonitorName проверяет и нормализует имя монитора.
func MonitorName(raw string) (string, error) {
	name := strings.TrimSpace(raw)

	if name == "" || utf8.RuneCountInString(name) > domain.MaxNameLength {
		return "", domain.ErrInvalidMonitorName
	}

	return name, nil
}

// MonitorTarget проверяет длину цели. Схема и допустимость адреса проверяются
// prober'ом (пакет infra/prober), чтобы правило жило в одном месте.
func MonitorTarget(raw string) (string, error) {
	target := strings.TrimSpace(raw)

	if target == "" || len(target) > domain.MaxTargetLength {
		return "", domain.ErrInvalidTarget
	}

	if !strings.HasPrefix(target, "http://") && !strings.HasPrefix(target, "https://") {
		return "", domain.ErrInvalidTarget
	}

	return target, nil
}

// MonitorKind проверяет вид проверки и что он реализован в текущей сборке.
func MonitorKind(raw string) (domain.MonitorKind, error) {
	kind := domain.MonitorKind(strings.TrimSpace(strings.ToLower(raw)))

	if !slices.Contains(domain.KnownMonitorKinds(), kind) {
		return "", domain.ErrInvalidMonitorKind
	}

	if !kind.IsImplemented() {
		return "", domain.ErrInvalidMonitorKind
	}

	return kind, nil
}

// Method проверяет и нормализует HTTP-метод проверки.
func Method(raw string) (string, error) {
	method := strings.ToUpper(strings.TrimSpace(raw))

	if !slices.Contains(allowedMethods, method) {
		return "", domain.ErrInvalidMethod
	}

	return method, nil
}

// Schedule проверяет интервал и таймаут вместе: таймаут обязан быть строго
// меньше интервала, иначе проверки будут наезжать друг на друга.
func Schedule(intervalSeconds, timeoutSeconds int) error {
	if intervalSeconds < domain.MinIntervalSeconds || intervalSeconds > domain.MaxIntervalSeconds {
		return domain.ErrInvalidInterval
	}

	if timeoutSeconds < domain.MinTimeoutSeconds || timeoutSeconds > domain.MaxTimeoutSeconds {
		return domain.ErrInvalidTimeout
	}

	if timeoutSeconds >= intervalSeconds {
		return domain.ErrTimeoutExceedsInterval
	}

	return nil
}

// ExpectedStatus проверяет ожидаемый код ответа.
func ExpectedStatus(code int) error {
	if code < MinStatusCode || code > MaxStatusCode {
		return domain.ErrInvalidExpectedStatus
	}

	return nil
}

// FailureThreshold проверяет порог падений до открытия инцидента.
func FailureThreshold(threshold int) error {
	if threshold < domain.MinThreshold || threshold > domain.MaxThreshold {
		return domain.ErrInvalidThreshold
	}

	return nil
}

// Paging проверяет и нормализует параметры страницы.
func Paging(limit, offset int) (int, int, error) {
	if limit == 0 {
		limit = DefaultPageLimit
	}

	if limit < 0 || limit > MaxPageLimit || offset < 0 {
		return 0, 0, domain.ErrInvalidPaging
	}

	return limit, offset, nil
}

// TimeRange проверяет интервал выборки статистики.
func TimeRange(from, to time.Time) error {
	if from.IsZero() || to.IsZero() || !from.Before(to) {
		return domain.ErrInvalidTimeRange
	}

	if to.Sub(from) > MaxTimeRange {
		return domain.ErrInvalidTimeRange
	}

	return nil
}

// UUID проверяет, что строка непуста (формат проверяет БД при приведении к UUID).
func UUID(id string) bool {
	return strings.TrimSpace(id) != ""
}
