package middleware

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
)

// routeUnmatched — метка для запросов, не попавших ни в один маршрут.
const routeUnmatched = "unmatched"

// RequestRecorder — метрики HTTP-запросов.
type RequestRecorder interface {
	RequestStarted()
	RequestFinished(method, route string, status int, duration time.Duration)
}

// Metrics считает запросы по шаблону маршрута chi, а не по пути: в пути стоят
// идентификаторы мониторов, и метка с ними дала бы серию на каждый монитор.
// Запись через defer: паника в обработчике не должна оставить запрос
// недосчитанным, а счётчик «в полёте» — задранным.
func Metrics(recorder RequestRecorder) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			started := time.Now()
			wrapped := chimiddleware.NewWrapResponseWriter(w, r.ProtoMajor)

			recorder.RequestStarted()

			defer func() {
				recorder.RequestFinished(
					r.Method, routePattern(r), statusOf(wrapped), time.Since(started),
				)
			}()

			next.ServeHTTP(wrapped, r)
		})
	}
}

func routePattern(r *http.Request) string {
	routeCtx := chi.RouteContext(r.Context())
	if routeCtx == nil {
		return routeUnmatched
	}

	if pattern := routeCtx.RoutePattern(); pattern != "" {
		return pattern
	}

	return routeUnmatched
}

// statusOf: обёртка chi возвращает 0, пока обработчик ничего не записал.
// На проводе в этом случае будет 200 — его и считаем.
func statusOf(wrapped chimiddleware.WrapResponseWriter) int {
	if status := wrapped.Status(); status != 0 {
		return status
	}

	return http.StatusOK
}
