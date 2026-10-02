package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	chimiddleware "github.com/go-chi/chi/v5/middleware"

	"github.com/St1lon/sentinel/internal/transport/http/apierrors"
	"github.com/St1lon/sentinel/internal/transport/http/dto"
)

const defaultStatsWindow = 24 * time.Hour

func (h *Handlers) writeJSON(w http.ResponseWriter, r *http.Request, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)

	if payload == nil {
		return
	}

	if err := json.NewEncoder(w).Encode(payload); err != nil {
		h.deps.Logger.ErrorContext(r.Context(), "encode response", slog.String("error", err.Error()))
	}
}

func (h *Handlers) writeError(w http.ResponseWriter, r *http.Request, err error) {
	apiErr := apierrors.From(err)

	if apiErr.Status >= http.StatusInternalServerError {
		h.deps.Logger.ErrorContext(r.Context(), "request failed",
			slog.String("error", err.Error()),
			slog.String("path", r.URL.Path),
			slog.String("request_id", chimiddleware.GetReqID(r.Context())),
		)
	}

	h.writeJSON(w, r, apiErr.Status, dto.ErrorResponse{
		Code:        apiErr.Code,
		Description: apiErr.Description,
	})
}

func (h *Handlers) writeAPIError(w http.ResponseWriter, r *http.Request, apiErr apierrors.APIError) {
	h.writeJSON(w, r, apiErr.Status, dto.ErrorResponse{
		Code:        apiErr.Code,
		Description: apiErr.Description,
	})
}

func (h *Handlers) decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	reader := http.MaxBytesReader(w, r.Body, h.deps.BodyLimit)

	decoder := json.NewDecoder(reader)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(dst); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			return apierrors.APIError{
				Status:      http.StatusRequestEntityTooLarge,
				Code:        apierrors.CodeRequestTooLarge,
				Description: "request body is too large",
			}
		}

		return apierrors.Malformed(err.Error())
	}

	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return apierrors.Malformed("body must contain a single json object")
	}

	return nil
}

func (h *Handlers) handleDecodeError(w http.ResponseWriter, r *http.Request, err error) {
	var apiErr apierrors.APIError
	if errors.As(err, &apiErr) {
		h.writeAPIError(w, r, apiErr)

		return
	}

	h.writeAPIError(w, r, apierrors.Malformed(err.Error()))
}

func intQuery(r *http.Request, name string, fallback int) (int, error) {
	raw := r.URL.Query().Get(name)
	if raw == "" {
		return fallback, nil
	}

	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0, apierrors.Malformed(name + " must be an integer")
	}

	return value, nil
}

func timeRangeQuery(r *http.Request, now time.Time) (time.Time, time.Time, error) {
	to := now
	from := now.Add(-defaultStatsWindow)

	if raw := r.URL.Query().Get("to"); raw != "" {
		parsed, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			return time.Time{}, time.Time{}, apierrors.Malformed("to must be an RFC3339 timestamp")
		}

		to = parsed
	}

	if raw := r.URL.Query().Get("from"); raw != "" {
		parsed, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			return time.Time{}, time.Time{}, apierrors.Malformed("from must be an RFC3339 timestamp")
		}

		from = parsed
	}

	return from.UTC(), to.UTC(), nil
}
