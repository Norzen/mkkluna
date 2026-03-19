package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"taskmanager/internal/middleware"
	"taskmanager/internal/model"
	"taskmanager/internal/service"
)

type TaskHandler struct {
	taskSvc service.TaskService
}

func NewTaskHandler(taskSvc service.TaskService) *TaskHandler {
	return &TaskHandler{taskSvc: taskSvc}
}

func (h *TaskHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	var req model.CreateTaskRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	task, err := h.taskSvc.Create(r.Context(), userID, req)
	if err != nil {
		if errors.Is(err, service.ErrNotMember) {
			respondError(w, http.StatusForbidden, err.Error())
			return
		}
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, task)
}

func (h *TaskHandler) List(w http.ResponseWriter, r *http.Request) {
	filter := model.TaskFilter{
		Page:     1,
		PageSize: 20,
	}

	if v := r.URL.Query().Get("team_id"); v != "" {
		if id, err := strconv.ParseInt(v, 10, 64); err == nil {
			filter.TeamID = id
		}
	}
	if v := r.URL.Query().Get("status"); v != "" {
		filter.Status = model.TaskStatus(v)
	}
	if v := r.URL.Query().Get("assignee_id"); v != "" {
		if id, err := strconv.ParseInt(v, 10, 64); err == nil {
			filter.AssigneeID = id
		}
	}
	if v := r.URL.Query().Get("page"); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			filter.Page = p
		}
	}
	if v := r.URL.Query().Get("page_size"); v != "" {
		if ps, err := strconv.Atoi(v); err == nil {
			filter.PageSize = ps
		}
	}

	resp, err := h.taskSvc.List(r.Context(), filter)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	respondJSON(w, http.StatusOK, resp)
}

func (h *TaskHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	taskIDStr := chi.URLParam(r, "id")
	taskID, err := strconv.ParseInt(taskIDStr, 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid task id")
		return
	}

	var req model.UpdateTaskRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	task, err := h.taskSvc.Update(r.Context(), userID, taskID, req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrTaskNotFound):
			respondError(w, http.StatusNotFound, err.Error())
		case errors.Is(err, service.ErrNotMember):
			respondError(w, http.StatusForbidden, err.Error())
		default:
			respondError(w, http.StatusBadRequest, err.Error())
		}
		return
	}

	respondJSON(w, http.StatusOK, task)
}

func (h *TaskHandler) GetHistory(w http.ResponseWriter, r *http.Request) {
	taskIDStr := chi.URLParam(r, "id")
	taskID, err := strconv.ParseInt(taskIDStr, 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid task id")
		return
	}

	history, err := h.taskSvc.GetHistory(r.Context(), taskID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	if history == nil {
		history = []model.TaskHistory{}
	}

	respondJSON(w, http.StatusOK, history)
}
