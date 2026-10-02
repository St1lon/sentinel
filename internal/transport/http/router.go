// Package http собирает HTTP-транспорт: роутер, цепочку middleware и сервер.
package http

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/St1lon/sentinel/internal/transport/http/apierrors"
	"github.com/St1lon/sentinel/internal/transport/http/dto"
	"github.com/St1lon/sentinel/internal/transport/http/handlers"
	"github.com/St1lon/sentinel/internal/transport/http/middleware"
)

// corsMaxAge — срок кэширования preflight-ответа браузером, в секундах.
const corsMaxAge = 300

// RouterDeps — зависимости роутера.
type RouterDeps struct {
	Handlers       *handlers.Handlers
	TokenParser    middleware.TokenParser
	Logger         *slog.Logger
	AllowedOrigins []string
}

// NewRouter собирает маршруты и цепочку middleware.
//
// Порядок цепочки: RequestID → RealIP → Recoverer → CORS → логгер.
// Логгер стоит последним из сквозных, чтобы в записи уже был request_id,
// а паника успевала превратиться в 500 до логирования.
func NewRouter(deps RouterDeps) http.Handler {
	router := chi.NewRouter()

	router.Use(chimiddleware.RequestID)
	router.Use(chimiddleware.RealIP)
	router.Use(chimiddleware.Recoverer)
	router.Use(cors.Handler(cors.Options{
		AllowedOrigins:   deps.AllowedOrigins,
		AllowedMethods:   []string{http.MethodGet, http.MethodPost, http.MethodPatch, http.MethodDelete, http.MethodOptions},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: false,
		MaxAge:           corsMaxAge,
	}))
	router.Use(middleware.Logger(deps.Logger))

	router.NotFound(notFound)
	router.MethodNotAllowed(methodNotAllowed)

	// Технические эндпоинты вне /api: их опрашивают балансировщик и оркестратор.
	router.Get("/healthz", deps.Handlers.Healthz)
	router.Get("/readyz", deps.Handlers.Readyz)

	router.Route("/api/v1", func(api chi.Router) {
		api.Post("/auth/register", deps.Handlers.Register)
		api.Post("/auth/login", deps.Handlers.Login)

		// Публичная статус-страница: без токена, доступ по знанию слага.
		api.Get("/public/status/{slug}", deps.Handlers.StatusPage)

		api.Group(func(protected chi.Router) {
			protected.Use(middleware.Auth(deps.TokenParser))

			protected.Get("/me", deps.Handlers.Me)

			protected.Route("/monitors", func(monitors chi.Router) {
				monitors.Post("/", deps.Handlers.CreateMonitor)
				monitors.Get("/", deps.Handlers.ListMonitors)

				monitors.Route("/{monitorID}", func(monitor chi.Router) {
					monitor.Get("/", deps.Handlers.GetMonitor)
					monitor.Patch("/", deps.Handlers.UpdateMonitor)
					monitor.Delete("/", deps.Handlers.DeleteMonitor)
					monitor.Get("/checks", deps.Handlers.ListChecks)
					monitor.Get("/stats", deps.Handlers.MonitorStats)
					monitor.Get("/incidents", deps.Handlers.ListIncidents)
				})
			})
		})
	})

	return router
}

func notFound(w http.ResponseWriter, _ *http.Request) {
	writeErrorResponse(w, http.StatusNotFound, apierrors.CodeRouteNotFound, "route not found")
}

func methodNotAllowed(w http.ResponseWriter, _ *http.Request) {
	writeErrorResponse(w, http.StatusMethodNotAllowed, apierrors.CodeMethodNotAllowed, "method not allowed")
}

func writeErrorResponse(w http.ResponseWriter, status int, code, description string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)

	_ = json.NewEncoder(w).Encode(dto.ErrorResponse{Code: code, Description: description})
}
