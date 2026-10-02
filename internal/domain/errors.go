package domain

import "errors"

var (
	ErrUserNotFound       = errors.New("user not found")
	ErrEmailAlreadyUsed   = errors.New("email already used")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidEmail       = errors.New("invalid email")
	ErrWeakPassword       = errors.New("password is too weak")
	ErrUnauthenticated    = errors.New("unauthenticated")

	ErrMonitorNotFound        = errors.New("monitor not found")
	ErrMonitorNameTaken       = errors.New("monitor name already taken")
	ErrInvalidMonitorName     = errors.New("invalid monitor name")
	ErrInvalidMonitorKind     = errors.New("invalid monitor kind")
	ErrInvalidTarget          = errors.New("invalid monitor target")
	ErrInvalidMethod          = errors.New("invalid http method")
	ErrInvalidInterval        = errors.New("invalid check interval")
	ErrInvalidTimeout         = errors.New("invalid check timeout")
	ErrTimeoutExceedsInterval = errors.New("timeout must be shorter than interval")
	ErrInvalidExpectedStatus  = errors.New("invalid expected status code")
	ErrInvalidThreshold       = errors.New("invalid failure threshold")
	ErrNothingToUpdate        = errors.New("nothing to update")

	ErrStatusPageNotFound = errors.New("status page not found")

	ErrInvalidTimeRange = errors.New("invalid time range")
	ErrInvalidPaging    = errors.New("invalid paging parameters")
	ErrInvalidBucket    = errors.New("invalid bucket size")

	ErrTargetNotAllowed = errors.New("target address is not allowed")
)
