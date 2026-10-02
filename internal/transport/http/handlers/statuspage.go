package handlers

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/St1lon/sentinel/internal/transport/http/mapper"
	getstatuspage "github.com/St1lon/sentinel/internal/usecase/statuspage/get"
)

// StatusPage — GET /api/v1/public/status/{slug}.
// Эндпоинт публичный: авторизация не требуется, доступ даёт знание слага.
func (h *Handlers) StatusPage(w http.ResponseWriter, r *http.Request) {
	now := time.Now().UTC()

	result, err := h.deps.StatusPage.Execute(r.Context(), &getstatuspage.Request{
		Slug: chi.URLParam(r, "slug"),
		Now:  now,
	})
	if err != nil {
		h.writeError(w, r, err)

		return
	}

	// Публичная страница кэшируется ненадолго: она читается чаще всего
	// и не содержит данных, зависящих от читателя.
	w.Header().Set("Cache-Control", "public, max-age=30")

	h.writeJSON(w, r, http.StatusOK, mapper.StatusPage(
		result.From, result.To, result.Monitors, result.Buckets, result.Incidents, now,
	))
}
