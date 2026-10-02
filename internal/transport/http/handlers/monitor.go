package handlers

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/St1lon/sentinel/internal/transport/http/dto"
	"github.com/St1lon/sentinel/internal/transport/http/mapper"
	createmonitor "github.com/St1lon/sentinel/internal/usecase/monitor/create"
	deletemonitor "github.com/St1lon/sentinel/internal/usecase/monitor/delete"
	getmonitor "github.com/St1lon/sentinel/internal/usecase/monitor/get"
	listmonitors "github.com/St1lon/sentinel/internal/usecase/monitor/list"
	monitorstats "github.com/St1lon/sentinel/internal/usecase/monitor/stats"
	updatemonitor "github.com/St1lon/sentinel/internal/usecase/monitor/update"
)

func (h *Handlers) CreateMonitor(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.userID(w, r)
	if !ok {
		return
	}

	var body dto.CreateMonitorRequest

	if err := h.decodeJSON(w, r, &body); err != nil {
		h.handleDecodeError(w, r, err)

		return
	}

	monitor, err := h.deps.CreateMonitor.Execute(r.Context(), &createmonitor.Request{
		UserID:           userID,
		Name:             body.Name,
		Kind:             body.Kind,
		Target:           body.Target,
		Method:           body.Method,
		IntervalSeconds:  body.IntervalSeconds,
		TimeoutSeconds:   body.TimeoutSeconds,
		ExpectedStatus:   body.ExpectedStatus,
		FailureThreshold: body.FailureThreshold,
		IsPublic:         boolOrFalse(body.IsPublic),
		Paused:           boolOrFalse(body.Paused),
	})
	if err != nil {
		h.writeError(w, r, err)

		return
	}

	h.writeJSON(w, r, http.StatusCreated, mapper.Monitor(monitor))
}

func (h *Handlers) ListMonitors(w http.ResponseWriter, r *http.Request) {
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

	result, err := h.deps.ListMonitors.Execute(r.Context(), &listmonitors.Request{
		UserID: userID,
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		h.writeError(w, r, err)

		return
	}

	h.writeJSON(w, r, http.StatusOK, dto.MonitorList{
		Items:  mapper.Monitors(result.Monitors),
		Total:  result.Total,
		Limit:  result.Limit,
		Offset: result.Offset,
	})
}

func (h *Handlers) GetMonitor(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.userID(w, r)
	if !ok {
		return
	}

	monitor, err := h.deps.GetMonitor.Execute(r.Context(), &getmonitor.Request{
		UserID:    userID,
		MonitorID: chi.URLParam(r, "monitorID"),
	})
	if err != nil {
		h.writeError(w, r, err)

		return
	}

	h.writeJSON(w, r, http.StatusOK, mapper.Monitor(monitor))
}

func (h *Handlers) UpdateMonitor(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.userID(w, r)
	if !ok {
		return
	}

	var body dto.UpdateMonitorRequest

	if err := h.decodeJSON(w, r, &body); err != nil {
		h.handleDecodeError(w, r, err)

		return
	}

	monitor, err := h.deps.UpdateMonitor.Execute(r.Context(), &updatemonitor.Request{
		UserID:           userID,
		MonitorID:        chi.URLParam(r, "monitorID"),
		Name:             body.Name,
		Target:           body.Target,
		Method:           body.Method,
		IntervalSeconds:  body.IntervalSeconds,
		TimeoutSeconds:   body.TimeoutSeconds,
		ExpectedStatus:   body.ExpectedStatus,
		FailureThreshold: body.FailureThreshold,
		IsPublic:         body.IsPublic,
		Paused:           body.Paused,
	})
	if err != nil {
		h.writeError(w, r, err)

		return
	}

	h.writeJSON(w, r, http.StatusOK, mapper.Monitor(monitor))
}

func (h *Handlers) DeleteMonitor(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.userID(w, r)
	if !ok {
		return
	}

	err := h.deps.DeleteMonitor.Execute(r.Context(), &deletemonitor.Request{
		UserID:    userID,
		MonitorID: chi.URLParam(r, "monitorID"),
	})
	if err != nil {
		h.writeError(w, r, err)

		return
	}

	h.writeJSON(w, r, http.StatusNoContent, nil)
}

func (h *Handlers) MonitorStats(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.userID(w, r)
	if !ok {
		return
	}

	from, to, err := timeRangeQuery(r, time.Now().UTC())
	if err != nil {
		h.handleDecodeError(w, r, err)

		return
	}

	result, err := h.deps.MonitorStats.Execute(r.Context(), &monitorstats.Request{
		UserID:    userID,
		MonitorID: chi.URLParam(r, "monitorID"),
		From:      from,
		To:        to,
		Bucket:    r.URL.Query().Get("bucket"),
	})
	if err != nil {
		h.writeError(w, r, err)

		return
	}

	h.writeJSON(w, r, http.StatusOK, mapper.MonitorStats(result.Stats, result.Buckets))
}

func boolOrFalse(value *bool) bool {
	return value != nil && *value
}
