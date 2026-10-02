package handlers

import (
	"net/http"

	"github.com/St1lon/sentinel/internal/transport/http/dto"
	"github.com/St1lon/sentinel/internal/transport/http/mapper"
	"github.com/St1lon/sentinel/internal/transport/http/middleware"
	loginuser "github.com/St1lon/sentinel/internal/usecase/auth/login"
	registeruser "github.com/St1lon/sentinel/internal/usecase/auth/register"
)

func (h *Handlers) Register(w http.ResponseWriter, r *http.Request) {
	var body dto.RegisterRequest

	if err := h.decodeJSON(w, r, &body); err != nil {
		h.handleDecodeError(w, r, err)

		return
	}

	result, err := h.deps.Register.Execute(r.Context(), &registeruser.Request{
		Email:    body.Email,
		Password: body.Password,
	})
	if err != nil {
		h.writeError(w, r, err)

		return
	}

	h.writeJSON(w, r, http.StatusCreated, dto.AuthResponse{
		Token:     result.Token,
		ExpiresAt: result.ExpiresAt,
		User:      mapper.User(result.User),
	})
}

func (h *Handlers) Login(w http.ResponseWriter, r *http.Request) {
	var body dto.LoginRequest

	if err := h.decodeJSON(w, r, &body); err != nil {
		h.handleDecodeError(w, r, err)

		return
	}

	result, err := h.deps.Login.Execute(r.Context(), &loginuser.Request{
		Email:    body.Email,
		Password: body.Password,
	})
	if err != nil {
		h.writeError(w, r, err)

		return
	}

	h.writeJSON(w, r, http.StatusOK, dto.AuthResponse{
		Token:     result.Token,
		ExpiresAt: result.ExpiresAt,
		User:      mapper.User(result.User),
	})
}

func (h *Handlers) userID(w http.ResponseWriter, r *http.Request) (string, bool) {
	userID, ok := middleware.UserIDFromRequest(r)
	if !ok {
		h.writeJSON(w, r, http.StatusUnauthorized, dto.ErrorResponse{
			Code:        "UNAUTHENTICATED",
			Description: "valid bearer token is required",
		})

		return "", false
	}

	return userID, true
}
