package middleware_test

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"runtime"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"

	"github.com/St1lon/sentinel/internal/transport/http/middleware"
)

func newChaosRouter(opts middleware.ChaosOptions) http.Handler {
	router := chi.NewRouter()
	router.Use(middleware.Chaos(opts, slog.New(slog.NewTextHandler(io.Discard, nil))))

	router.Get("/healthz", func(http.ResponseWriter, *http.Request) {})
	router.Get("/readyz", func(http.ResponseWriter, *http.Request) {})
	router.Get("/api/v1/monitors", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	return router
}

func chaosStatus(t *testing.T, handler http.Handler, path string) int {
	t.Helper()

	recorder := httptest.NewRecorder()
	request := httptest.NewRequestWithContext(context.Background(), http.MethodGet, path, nil)
	handler.ServeHTTP(recorder, request)

	return recorder.Code
}

func TestChaos_ErrorRateOneFailsEveryAPIRequest(t *testing.T) {
	t.Parallel()

	router := newChaosRouter(middleware.ChaosOptions{ErrorRate: 1})

	for range 5 {
		require.Equal(t, http.StatusServiceUnavailable, chaosStatus(t, router, "/api/v1/monitors"))
	}
}

// Сбои не должны попадать в проверки живости: 503 на /healthz перезапустит
// под, и демонстрация деградации превратится в падение.
func TestChaos_ProbesStayHealthy(t *testing.T) {
	t.Parallel()

	router := newChaosRouter(middleware.ChaosOptions{ErrorRate: 1})

	require.Equal(t, http.StatusOK, chaosStatus(t, router, "/healthz"))
	require.Equal(t, http.StatusOK, chaosStatus(t, router, "/readyz"))
}

func TestChaos_ReadyzFailAffectsReadinessOnly(t *testing.T) {
	t.Parallel()

	router := newChaosRouter(middleware.ChaosOptions{ReadyzFail: true})

	require.Equal(t, http.StatusServiceUnavailable, chaosStatus(t, router, "/readyz"))
	require.Equal(t, http.StatusOK, chaosStatus(t, router, "/healthz"))
	require.Equal(t, http.StatusNoContent, chaosStatus(t, router, "/api/v1/monitors"))
}

func TestChaos_ZeroRatesPassEverythingThrough(t *testing.T) {
	t.Parallel()

	router := newChaosRouter(middleware.ChaosOptions{})

	require.Equal(t, http.StatusNoContent, chaosStatus(t, router, "/api/v1/monitors"))
	require.Equal(t, http.StatusOK, chaosStatus(t, router, "/healthz"))
}

func TestChaos_LatencyDelaysAPIRequest(t *testing.T) {
	t.Parallel()

	router := newChaosRouter(middleware.ChaosOptions{
		Latency:     120 * time.Millisecond,
		LatencyRate: 1,
	})

	started := time.Now()
	require.Equal(t, http.StatusNoContent, chaosStatus(t, router, "/api/v1/monitors"))
	require.GreaterOrEqual(t, time.Since(started), 100*time.Millisecond)

	// Задержка тоже не касается проверок живости.
	started = time.Now()
	require.Equal(t, http.StatusOK, chaosStatus(t, router, "/healthz"))
	require.Less(t, time.Since(started), 100*time.Millisecond)
}

// Имитация утечки горутин: каждый /api/ запрос оставляет после себя горутины,
// проверки живости — нет. Без t.Parallel: утечка копится в состоянии пакета,
// и параллельный запуск исказил бы счётчики.
func TestChaos_GoroutineLeakSpawnsGoroutines(t *testing.T) {
	router := newChaosRouter(middleware.ChaosOptions{GoroutineLeak: 10})

	before := runtime.NumGoroutine()
	for range 3 {
		require.Equal(t, http.StatusNoContent, chaosStatus(t, router, "/api/v1/monitors"))
	}
	require.GreaterOrEqual(t, runtime.NumGoroutine(), before+25,
		"каждый из трёх запросов должен оставить по 10 горутин")

	healthBefore := runtime.NumGoroutine()
	require.Equal(t, http.StatusOK, chaosStatus(t, router, "/healthz"))
	require.LessOrEqual(t, runtime.NumGoroutine(), healthBefore+2,
		"проверки живости горутины не текут")
}

// Прерванный запрос не доходит до маршрутизации chi, и метрика без помощи
// припишет его метке unmatched — а она из SLI доступности исключена.
func TestChaos_KeepsRoutePatternOnInjectedError(t *testing.T) {
	t.Parallel()

	var pattern string

	router := chi.NewRouter()
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r)

			pattern = chi.RouteContext(r.Context()).RoutePattern()
		})
	})
	router.Use(middleware.Chaos(
		middleware.ChaosOptions{ErrorRate: 1},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
	))
	router.Get("/api/v1/monitors/{monitorID}", func(http.ResponseWriter, *http.Request) {})

	require.Equal(t,
		http.StatusServiceUnavailable,
		chaosStatus(t, router, "/api/v1/monitors/42"),
	)
	require.Equal(t, "/api/v1/monitors/{monitorID}", pattern)
}
