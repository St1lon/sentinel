package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"

	"github.com/St1lon/sentinel/internal/transport/http/middleware"
)

type recordedRequest struct {
	method   string
	route    string
	status   int
	duration time.Duration
}

type recorderStub struct {
	started  int
	finished []recordedRequest
}

func (s *recorderStub) RequestStarted() {
	s.started++
}

func (s *recorderStub) RequestFinished(method, route string, status int, duration time.Duration) {
	s.finished = append(s.finished, recordedRequest{method, route, status, duration})
}

func newRouter(recorder middleware.RequestRecorder) http.Handler {
	router := chi.NewRouter()
	router.Use(middleware.Metrics(recorder))

	router.Get("/api/v1/monitors/{monitorID}", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	router.Get("/healthz", func(http.ResponseWriter, *http.Request) {})

	router.Get("/boom", func(http.ResponseWriter, *http.Request) {
		panic("boom")
	})

	return router
}

func TestMetrics_UsesRoutePatternInsteadOfPath(t *testing.T) {
	t.Parallel()

	recorder := &recorderStub{}
	request := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/v1/monitors/11111111-1111-1111-1111-111111111111", nil)

	newRouter(recorder).ServeHTTP(httptest.NewRecorder(), request)

	require.Len(t, recorder.finished, 1)
	require.Equal(t, 1, recorder.started)
	require.Equal(t, "/api/v1/monitors/{monitorID}", recorder.finished[0].route,
		"идентификатор монитора не должен попадать в метку")
	require.Equal(t, http.StatusNoContent, recorder.finished[0].status)
}

func TestMetrics_UnmatchedRouteGetsOwnLabel(t *testing.T) {
	t.Parallel()

	recorder := &recorderStub{}
	request := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/no-such-route", nil)

	newRouter(recorder).ServeHTTP(httptest.NewRecorder(), request)

	require.Len(t, recorder.finished, 1)
	require.Equal(t, "unmatched", recorder.finished[0].route)
	require.Equal(t, http.StatusNotFound, recorder.finished[0].status)
}

func TestMetrics_HandlerWithoutWriteCountsAs200(t *testing.T) {
	t.Parallel()

	recorder := &recorderStub{}
	request := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/healthz", nil)

	newRouter(recorder).ServeHTTP(httptest.NewRecorder(), request)

	require.Len(t, recorder.finished, 1)
	require.Equal(t, http.StatusOK, recorder.finished[0].status)
}

// Паника не должна оставлять запрос недосчитанным: иначе счётчик запросов
// «в полёте» уползёт вверх навсегда.
func TestMetrics_PanicIsStillCounted(t *testing.T) {
	t.Parallel()

	recorder := &recorderStub{}
	request := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/boom", nil)

	require.Panics(t, func() {
		newRouter(recorder).ServeHTTP(httptest.NewRecorder(), request)
	})

	require.Len(t, recorder.finished, 1)
	require.Equal(t, "/boom", recorder.finished[0].route)
}
