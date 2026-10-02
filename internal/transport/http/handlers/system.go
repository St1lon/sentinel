package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/St1lon/sentinel/internal/transport/http/apierrors"
	"github.com/St1lon/sentinel/internal/transport/http/dto"
	"github.com/St1lon/sentinel/internal/transport/http/mapper"
	getuser "github.com/St1lon/sentinel/internal/usecase/user/get"
)

const readinessTimeout = 2 * time.Second

func (h *Handlers) Healthz(w http.ResponseWriter, r *http.Request) {
	h.writeJSON(w, r, http.StatusOK, dto.Health{
		Status:  "ok",
		Version: h.deps.Version,
		Env:     h.deps.Env,
	})
}

func (h *Handlers) Readyz(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), readinessTimeout)
	defer cancel()

	if err := h.deps.DB.Ping(ctx); err != nil {
		h.writeAPIError(w, r, apierrors.APIError{
			Status:      http.StatusServiceUnavailable,
			Code:        apierrors.CodeServiceUnavailable,
			Description: "database is not reachable",
		})

		return
	}

	h.writeJSON(w, r, http.StatusOK, dto.Health{
		Status:  "ready",
		Version: h.deps.Version,
		Env:     h.deps.Env,
	})
}

func (h *Handlers) Me(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.userID(w, r)
	if !ok {
		return
	}

	user, err := h.deps.CurrentUser.Execute(r.Context(), &getuser.Request{UserID: userID})
	if err != nil {
		h.writeError(w, r, err)

		return
	}

	h.writeJSON(w, r, http.StatusOK, mapper.User(user))
}
