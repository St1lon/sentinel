package prober_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/St1lon/sentinel/internal/domain"
	"github.com/St1lon/sentinel/internal/infra/prober"
)

// localProber разрешает приватные адреса: тестовый сервер поднимается на 127.0.0.1.
func localProber() *prober.HTTPProber {
	return prober.NewHTTPProber(prober.Options{
		UserAgent:        "SentinelTest/1.0",
		MaxResponseBytes: 4096,
		MaxRedirects:     2,
		AllowPrivate:     true,
	})
}

func monitorFor(target string, expected int) *domain.Monitor {
	return &domain.Monitor{
		ID:             "11111111-1111-1111-1111-111111111111",
		Kind:           domain.MonitorKindHTTP,
		Target:         target,
		Method:         http.MethodGet,
		TimeoutSeconds: 5,
		ExpectedStatus: expected,
	}
}

func TestHTTPProber_SuccessOnExpectedStatus(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "SentinelTest/1.0", r.Header.Get("User-Agent"))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	check, err := localProber().Probe(context.Background(), monitorFor(server.URL, http.StatusOK))
	require.NoError(t, err)
	require.True(t, check.Up)
	require.Equal(t, http.StatusOK, *check.StatusCode)
	require.Nil(t, check.Error)
	require.GreaterOrEqual(t, check.LatencyMS, 0)
}

func TestHTTPProber_FailsOnUnexpectedStatus(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	check, err := localProber().Probe(context.Background(), monitorFor(server.URL, http.StatusOK))
	require.NoError(t, err, "неуспешная проверка — это результат, а не ошибка")
	require.False(t, check.Up)
	require.Equal(t, http.StatusServiceUnavailable, *check.StatusCode)
	require.Equal(t, "unexpected status 503, expected 200", *check.Error)
}

func TestHTTPProber_CustomExpectedStatusMatches(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	check, err := localProber().Probe(context.Background(), monitorFor(server.URL, http.StatusNoContent))
	require.NoError(t, err)
	require.True(t, check.Up, "ожидаемым можно объявить любой код")
}

func TestHTTPProber_FailsOnUnreachableTarget(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	url := server.URL
	server.Close() // порт закрыт — соединение не установится

	check, err := localProber().Probe(context.Background(), monitorFor(url, http.StatusOK))
	require.NoError(t, err)
	require.False(t, check.Up)
	require.Nil(t, check.StatusCode)
	require.NotNil(t, check.Error)
}

func TestHTTPProber_BlockedTargetBecomesFailedCheck(t *testing.T) {
	t.Parallel()

	// Prober в обычном режиме: приватные адреса запрещены.
	strict := prober.NewHTTPProber(prober.Options{
		UserAgent:        "SentinelTest/1.0",
		MaxResponseBytes: 4096,
		AllowPrivate:     false,
	})

	check, err := strict.Probe(context.Background(), monitorFor("http://169.254.169.254/", http.StatusOK))

	// Монитор существует, поэтому это не ошибка выполнения, а проваленная
	// проверка с понятной причиной — пользователь увидит её на странице.
	require.NoError(t, err)
	require.False(t, check.Up)
	require.Contains(t, *check.Error, "not allowed")
}

func TestHTTPProber_RejectsUnimplementedKind(t *testing.T) {
	t.Parallel()

	monitor := monitorFor("https://example.com", http.StatusOK)
	monitor.Kind = domain.MonitorKindTLSCert

	_, err := localProber().Probe(context.Background(), monitor)
	require.ErrorIs(t, err, domain.ErrInvalidMonitorKind)
}

func TestHTTPProber_FollowsRedirectsWithinLimit(t *testing.T) {
	t.Parallel()

	var server *httptest.Server

	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/final" {
			w.WriteHeader(http.StatusOK)

			return
		}

		http.Redirect(w, r, server.URL+"/final", http.StatusFound)
	}))
	defer server.Close()

	check, err := localProber().Probe(context.Background(), monitorFor(server.URL+"/start", http.StatusOK))
	require.NoError(t, err)
	require.True(t, check.Up)
}

func TestHTTPProber_StopsAfterTooManyRedirects(t *testing.T) {
	t.Parallel()

	var server *httptest.Server

	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, server.URL+"/loop", http.StatusFound)
	}))
	defer server.Close()

	check, err := localProber().Probe(context.Background(), monitorFor(server.URL, http.StatusOK))
	require.NoError(t, err)
	require.False(t, check.Up)
	require.Contains(t, *check.Error, "redirects")
}
