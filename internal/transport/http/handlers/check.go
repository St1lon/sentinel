package handlers

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/St1lon/sentinel/internal/transport/http/dto"
	"github.com/St1lon/sentinel/internal/transport/http/mapper"
	listchecks "github.com/St1lon/sentinel/internal/usecase/check/list"
	listincidents "github.com/St1lon/sentinel/internal/usecase/incident/list"
)

func (h *Handlers) ListChecks(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.userID(w, r)
	if !ok {
		return
	}

	from, to, err := timeRangeQuery(r, time.Now().UTC())
	if err != nil {
		h.handleDecodeError(w, r, err)

		return
	}

	limit, err := intQuery(r, "limit", 0)
	if err != nil {
		h.handleDecodeError(w, r, err)

		return
	}

	result, err := h.deps.ListChecks.Execute(r.Context(), &listchecks.Request{
		UserID:    userID,
		MonitorID: chi.URLParam(r, "monitorID"),
		From:      from,
		To:        to,
		Limit:     limit,
	})
	if err != nil {
		h.writeError(w, r, err)

		return
	}

	h.writeJSON(w, r, http.StatusOK, dto.CheckList{
		MonitorID: result.Monitor.ID,
		Items:     mapper.Checks(result.Checks),
	})
}

func (h *Handlers) ListIncidents(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.userID(w, r)
	if !ok {
		return
	}

	limit, err := intQuery(r, "limit", 0)
	if err != nil {
		h.handleDecodeError(w, r, err)

		return
	}

	offset, err := intQuery(r, "offset", 0)
	if err != nil {
		h.handleDecodeError(w, r, err)

		return
	}

	result, err := h.deps.ListIncidents.Execute(r.Context(), &listincidents.Request{
		UserID:    userID,
		MonitorID: chi.URLParam(r, "monitorID"),
		Limit:     limit,
		Offset:    offset,
	})
	if err != nil {
		h.writeError(w, r, err)

		return
	}

	h.writeJSON(w, r, http.StatusOK, dto.IncidentList{
		Items:  mapper.Incidents(result.Incidents, time.Now().UTC()),
		Total:  result.Total,
		Limit:  result.Limit,
		Offset: result.Offset,
	})
}
