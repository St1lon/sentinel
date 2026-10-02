package apierrors_test

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/St1lon/sentinel/internal/domain"
	"github.com/St1lon/sentinel/internal/transport/http/apierrors"
)

func TestFrom_MapsDomainErrors(t *testing.T) {
	t.Parallel()

	cases := []struct {
		err    error
		status int
		code   string
	}{
		{domain.ErrInvalidEmail, http.StatusBadRequest, apierrors.CodeInvalidEmail},
		{domain.ErrWeakPassword, http.StatusBadRequest, apierrors.CodeWeakPassword},
		{domain.ErrEmailAlreadyUsed, http.StatusConflict, apierrors.CodeEmailAlreadyUsed},
		{domain.ErrInvalidCredentials, http.StatusUnauthorized, apierrors.CodeInvalidCredentials},
		{domain.ErrUnauthenticated, http.StatusUnauthorized, apierrors.CodeUnauthenticated},
		{domain.ErrMonitorNotFound, http.StatusNotFound, apierrors.CodeMonitorNotFound},
		{domain.ErrMonitorNameTaken, http.StatusConflict, apierrors.CodeMonitorNameTaken},
		{domain.ErrTargetNotAllowed, http.StatusBadRequest, apierrors.CodeTargetNotAllowed},
		{domain.ErrTimeoutExceedsInterval, http.StatusBadRequest, apierrors.CodeInvalidSchedule},
		{domain.ErrStatusPageNotFound, http.StatusNotFound, apierrors.CodeStatusPageNotFound},
		{domain.ErrInvalidBucket, http.StatusBadRequest, apierrors.CodeInvalidBucket},
		{domain.ErrNothingToUpdate, http.StatusBadRequest, apierrors.CodeNothingToUpdate},
	}

	for _, testCase := range cases {
		t.Run(testCase.code, func(t *testing.T) {
			t.Parallel()

			apiErr := apierrors.From(testCase.err)
			require.Equal(t, testCase.status, apiErr.Status)
			require.Equal(t, testCase.code, apiErr.Code)
		})
	}
}

func TestFrom_UnwrapsWrappedErrors(t *testing.T) {
	t.Parallel()

	// Репозиторий оборачивает ошибки контекстом, маппинг обязан это переживать.
	wrapped := fmt.Errorf("select monitor: %w", domain.ErrMonitorNotFound)

	apiErr := apierrors.From(wrapped)
	require.Equal(t, http.StatusNotFound, apiErr.Status)
	require.Equal(t, apierrors.CodeMonitorNotFound, apiErr.Code)
}

func TestFrom_UnknownErrorHidesDetails(t *testing.T) {
	t.Parallel()

	// Внутренняя ошибка не должна раскрывать клиенту устройство сервиса:
	// подробности уходят в лог, наружу — нейтральное сообщение.
	apiErr := apierrors.From(errors.New(`pq: relation "monitors" does not exist`))

	require.Equal(t, http.StatusInternalServerError, apiErr.Status)
	require.Equal(t, apierrors.CodeInternal, apiErr.Code)
	require.Equal(t, "internal error", apiErr.Description)
	require.NotContains(t, apiErr.Description, "monitors")
}

func TestFrom_UserNotFoundLooksLikeUnauthenticated(t *testing.T) {
	t.Parallel()

	// Токен есть, а пользователя уже нет — это 401, а не 404:
	// наружу не должно утекать, что аккаунт удалён.
	apiErr := apierrors.From(domain.ErrUserNotFound)
	require.Equal(t, http.StatusUnauthorized, apiErr.Status)
}

func TestIsInternal(t *testing.T) {
	t.Parallel()

	require.True(t, apierrors.IsInternal(errors.New("boom")))
	require.False(t, apierrors.IsInternal(domain.ErrMonitorNotFound))
}

func TestAPIErrorImplementsError(t *testing.T) {
	t.Parallel()

	var err error = apierrors.Malformed("body must be json")

	var apiErr apierrors.APIError
	require.ErrorAs(t, err, &apiErr)
	require.Equal(t, http.StatusBadRequest, apiErr.Status)
	require.Equal(t, apierrors.CodeMalformedBody, apiErr.Code)
}
