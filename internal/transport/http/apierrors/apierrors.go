package apierrors

import (
	"errors"
	"net/http"

	"github.com/St1lon/sentinel/internal/domain"
)

const (
	CodeInvalidEmail       = "INVALID_EMAIL"
	CodeWeakPassword       = "WEAK_PASSWORD"
	CodeEmailAlreadyUsed   = "EMAIL_ALREADY_USED"
	CodeInvalidCredentials = "INVALID_CREDENTIALS"
	CodeUnauthenticated    = "UNAUTHENTICATED"

	CodeMonitorNotFound    = "MONITOR_NOT_FOUND"
	CodeMonitorNameTaken   = "MONITOR_NAME_TAKEN"
	CodeInvalidMonitorName = "INVALID_MONITOR_NAME"
	CodeInvalidMonitorKind = "INVALID_MONITOR_KIND"
	CodeInvalidTarget      = "INVALID_TARGET"
	CodeInvalidMethod      = "INVALID_METHOD"
	CodeInvalidInterval    = "INVALID_INTERVAL"
	CodeInvalidTimeout     = "INVALID_TIMEOUT"
	CodeInvalidSchedule    = "INVALID_SCHEDULE"
	CodeInvalidStatusCode  = "INVALID_EXPECTED_STATUS"
	CodeInvalidThreshold   = "INVALID_FAILURE_THRESHOLD"
	CodeNothingToUpdate    = "NOTHING_TO_UPDATE"
	CodeTargetNotAllowed   = "TARGET_NOT_ALLOWED"
	CodeStatusPageNotFound = "STATUS_PAGE_NOT_FOUND"
	CodeInvalidTimeRange   = "INVALID_TIME_RANGE"
	CodeInvalidPaging      = "INVALID_PAGING"
	CodeInvalidBucket      = "INVALID_BUCKET"
	CodeMalformedBody      = "MALFORMED_BODY"
	CodeRequestTooLarge    = "REQUEST_TOO_LARGE"
	CodeMethodNotAllowed   = "METHOD_NOT_ALLOWED"
	CodeRouteNotFound      = "ROUTE_NOT_FOUND"
	CodeInternal           = "INTERNAL_ERROR"
	CodeServiceUnavailable = "SERVICE_UNAVAILABLE"
)

type APIError struct {
	Status      int
	Code        string
	Description string
}

func (e APIError) Error() string {
	return e.Code + ": " + e.Description
}

var mapping = []struct {
	err    error
	status int
	code   string
}{
	{domain.ErrInvalidEmail, http.StatusBadRequest, CodeInvalidEmail},
	{domain.ErrWeakPassword, http.StatusBadRequest, CodeWeakPassword},
	{domain.ErrEmailAlreadyUsed, http.StatusConflict, CodeEmailAlreadyUsed},
	{domain.ErrInvalidCredentials, http.StatusUnauthorized, CodeInvalidCredentials},
	{domain.ErrUnauthenticated, http.StatusUnauthorized, CodeUnauthenticated},
	{domain.ErrUserNotFound, http.StatusUnauthorized, CodeUnauthenticated},

	{domain.ErrMonitorNotFound, http.StatusNotFound, CodeMonitorNotFound},
	{domain.ErrMonitorNameTaken, http.StatusConflict, CodeMonitorNameTaken},
	{domain.ErrInvalidMonitorName, http.StatusBadRequest, CodeInvalidMonitorName},
	{domain.ErrInvalidMonitorKind, http.StatusBadRequest, CodeInvalidMonitorKind},
	{domain.ErrInvalidTarget, http.StatusBadRequest, CodeInvalidTarget},
	{domain.ErrTargetNotAllowed, http.StatusBadRequest, CodeTargetNotAllowed},
	{domain.ErrInvalidMethod, http.StatusBadRequest, CodeInvalidMethod},
	{domain.ErrInvalidInterval, http.StatusBadRequest, CodeInvalidInterval},
	{domain.ErrInvalidTimeout, http.StatusBadRequest, CodeInvalidTimeout},
	{domain.ErrTimeoutExceedsInterval, http.StatusBadRequest, CodeInvalidSchedule},
	{domain.ErrInvalidExpectedStatus, http.StatusBadRequest, CodeInvalidStatusCode},
	{domain.ErrInvalidThreshold, http.StatusBadRequest, CodeInvalidThreshold},
	{domain.ErrNothingToUpdate, http.StatusBadRequest, CodeNothingToUpdate},

	{domain.ErrStatusPageNotFound, http.StatusNotFound, CodeStatusPageNotFound},
	{domain.ErrInvalidTimeRange, http.StatusBadRequest, CodeInvalidTimeRange},
	{domain.ErrInvalidPaging, http.StatusBadRequest, CodeInvalidPaging},
	{domain.ErrInvalidBucket, http.StatusBadRequest, CodeInvalidBucket},
}

func From(err error) APIError {
	for _, item := range mapping {
		if errors.Is(err, item.err) {
			return APIError{Status: item.status, Code: item.code, Description: err.Error()}
		}
	}

	return APIError{
		Status:      http.StatusInternalServerError,
		Code:        CodeInternal,
		Description: "internal error",
	}
}

func IsInternal(err error) bool {
	return From(err).Status >= http.StatusInternalServerError
}

func Malformed(description string) APIError {
	return APIError{Status: http.StatusBadRequest, Code: CodeMalformedBody, Description: description}
}
