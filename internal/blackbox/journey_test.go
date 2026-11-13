package blackbox_test

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/St1lon/sentinel/internal/blackbox"
)

const (
	testSlug     = "demo-slug"
	testToken    = "test-token"
	testEmail    = "demo@sentinel.local"
	testPassword = "demo-password-123"
)

type recordedStep struct {
	success  bool
	duration time.Duration
}

type metricsStub struct {
	steps     map[string]recordedStep
	journeys  []bool
	stepCalls int
}

func newMetricsStub() *metricsStub {
	return &metricsStub{steps: make(map[string]recordedStep)}
}

func (s *metricsStub) StepFinished(step string, success bool, duration time.Duration) {
	s.steps[step] = recordedStep{success: success, duration: duration}
	s.stepCalls++
}

func (s *metricsStub) JourneyFinished(success bool) {
	s.journeys = append(s.journeys, success)
}

// handlers — ответы API по путям сценария. Тест подменяет нужный путь,
// остальные отвечают как здоровый сервис.
type handlers map[string]http.HandlerFunc

func newServer(t *testing.T, overrides handlers) *httptest.Server {
	t.Helper()

	defaults := handlers{
		"/healthz": okJSON(`{"status":"ok"}`),
		"/readyz":  okJSON(`{"status":"ready"}`),
		"/api/v1/auth/login": okJSON(
			`{"token":"` + testToken + `","user":{"status_page_slug":"` + testSlug + `"}}`,
		),
		"/api/v1/monitors": func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("Authorization") != "Bearer "+testToken {
				w.WriteHeader(http.StatusUnauthorized)

				return
			}

			okJSON(`{"items":[]}`)(w, r)
		},
		"/api/v1/public/status/" + testSlug: okJSON(`{"monitors":[]}`),
	}

	for path, handler := range overrides {
		defaults[path] = handler
	}

	mux := http.NewServeMux()
	for path, handler := range defaults {
		mux.HandleFunc(path, handler)
	}

	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	return server
}

func okJSON(body string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, body)
	}
}

func newJourney(target string, metrics blackbox.Metrics, slug string) *blackbox.Journey {
	return blackbox.New(metrics, slog.New(slog.NewTextHandler(io.Discard, nil)), blackbox.Options{
		TargetURL:     target,
		Interval:      time.Second,
		Timeout:       time.Second,
		LoginEmail:    testEmail,
		LoginPassword: testPassword,
		StatusSlug:    slug,
		UserAgent:     "SentinelProber/test",
	})
}

func TestRunOnce_AllStepsPass(t *testing.T) {
	t.Parallel()

	metrics := newMetricsStub()
	server := newServer(t, nil)

	require.True(t, newJourney(server.URL, metrics, testSlug).RunOnce(context.Background()))

	for _, step := range []string{
		blackbox.StepHealthz, blackbox.StepReadyz, blackbox.StepLogin,
		blackbox.StepMonitors, blackbox.StepStatusPage,
	} {
		require.True(t, metrics.steps[step].success, "шаг %s должен быть успешным", step)
	}

	require.Equal(t, []bool{true}, metrics.journeys)
}

func TestRunOnce_FailedStepFailsJourneyOnly(t *testing.T) {
	t.Parallel()

	metrics := newMetricsStub()
	server := newServer(t, handlers{
		"/api/v1/monitors": func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		},
	})

	require.False(t, newJourney(server.URL, metrics, testSlug).RunOnce(context.Background()))

	require.False(t, metrics.steps[blackbox.StepMonitors].success)
	require.True(t, metrics.steps[blackbox.StepHealthz].success, "остальные шаги не должны пострадать")
	require.True(t, metrics.steps[blackbox.StepLogin].success)
	require.Equal(t, []bool{false}, metrics.journeys)
}

// Вход не прошёл — списка мониторов пользователь тоже не увидит. Шаг обязан
// быть записан как неуспешный, а не пропущен: иначе его gauge застынет
// на последнем удачном значении.
func TestRunOnce_LoginFailureMarksMonitorsStepFailed(t *testing.T) {
	t.Parallel()

	metrics := newMetricsStub()
	server := newServer(t, handlers{
		"/api/v1/auth/login": func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
		},
	})

	require.False(t, newJourney(server.URL, metrics, testSlug).RunOnce(context.Background()))

	require.False(t, metrics.steps[blackbox.StepLogin].success)
	require.False(t, metrics.steps[blackbox.StepMonitors].success)
}

func TestRunOnce_LoginWithoutTokenIsFailure(t *testing.T) {
	t.Parallel()

	metrics := newMetricsStub()
	server := newServer(t, handlers{
		"/api/v1/auth/login": okJSON(`{"user":{"id":"1"}}`),
	})

	require.False(t, newJourney(server.URL, metrics, testSlug).RunOnce(context.Background()))
	require.False(t, metrics.steps[blackbox.StepLogin].success, "ответ 200 без токена — тоже отказ шага")
}

func TestRunOnce_EmptySlugSkipsStatusPage(t *testing.T) {
	t.Parallel()

	metrics := newMetricsStub()
	server := newServer(t, nil)

	require.True(t, newJourney(server.URL, metrics, "").RunOnce(context.Background()))

	_, recorded := metrics.steps[blackbox.StepStatusPage]
	require.False(t, recorded, "без slug шаг не выполняется и метрики не портит")
	require.Equal(t, 4, metrics.stepCalls)
}

func TestRunOnce_SlowResponseIsFailure(t *testing.T) {
	t.Parallel()

	metrics := newMetricsStub()
	server := newServer(t, handlers{
		"/healthz": func(w http.ResponseWriter, _ *http.Request) {
			time.Sleep(300 * time.Millisecond)
			okJSON(`{"status":"ok"}`)(w, nil)
		},
	})

	journey := blackbox.New(metrics, slog.New(slog.NewTextHandler(io.Discard, nil)), blackbox.Options{
		TargetURL:     server.URL,
		Interval:      time.Second,
		Timeout:       50 * time.Millisecond,
		LoginEmail:    testEmail,
		LoginPassword: testPassword,
		UserAgent:     "SentinelProber/test",
	})

	require.False(t, journey.RunOnce(context.Background()))
	require.False(t, metrics.steps[blackbox.StepHealthz].success, "таймаут шага — отказ")
}

func TestRun_StopsOnContextCancel(t *testing.T) {
	t.Parallel()

	metrics := newMetricsStub()
	server := newServer(t, nil)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	require.NoError(t, newJourney(server.URL, metrics, testSlug).Run(ctx))
	require.Len(t, metrics.journeys, 1, "первый прогон идёт до тика, дальше цикл выходит")
}
