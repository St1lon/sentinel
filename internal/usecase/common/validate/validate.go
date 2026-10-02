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
	MinPasswordLength = 8
	MaxPasswordLength = 72
	MaxEmailLength    = 254

	MinStatusCode = 100
	MaxStatusCode = 599

	MaxPageLimit     = 200
	DefaultPageLimit = 50
	MaxTimeRange     = 400 * 24 * time.Hour
)

var allowedMethods = []string{"GET", "HEAD", "OPTIONS"}

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

func Password(password string) error {
	if len(password) < MinPasswordLength || len(password) > MaxPasswordLength {
		return domain.ErrWeakPassword
	}

	return nil
}

func MonitorName(raw string) (string, error) {
	name := strings.TrimSpace(raw)

	if name == "" || utf8.RuneCountInString(name) > domain.MaxNameLength {
		return "", domain.ErrInvalidMonitorName
	}

	return name, nil
}

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

func Method(raw string) (string, error) {
	method := strings.ToUpper(strings.TrimSpace(raw))

	if !slices.Contains(allowedMethods, method) {
		return "", domain.ErrInvalidMethod
	}

	return method, nil
}

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

func ExpectedStatus(code int) error {
	if code < MinStatusCode || code > MaxStatusCode {
		return domain.ErrInvalidExpectedStatus
	}

	return nil
}

func FailureThreshold(threshold int) error {
	if threshold < domain.MinThreshold || threshold > domain.MaxThreshold {
		return domain.ErrInvalidThreshold
	}

	return nil
}

func Paging(limit, offset int) (int, int, error) {
	if limit == 0 {
		limit = DefaultPageLimit
	}

	if limit < 0 || limit > MaxPageLimit || offset < 0 {
		return 0, 0, domain.ErrInvalidPaging
	}

	return limit, offset, nil
}

func TimeRange(from, to time.Time) error {
	if from.IsZero() || to.IsZero() || !from.Before(to) {
		return domain.ErrInvalidTimeRange
	}

	if to.Sub(from) > MaxTimeRange {
		return domain.ErrInvalidTimeRange
	}

	return nil
}

func UUID(id string) bool {
	return strings.TrimSpace(id) != ""
}
