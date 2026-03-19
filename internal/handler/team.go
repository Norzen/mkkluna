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

type TeamHandler struct {
	teamSvc service.TeamService
}

func NewTeamHandler(teamSvc service.TeamService) *TeamHandler {
	return &TeamHandler{teamSvc: teamSvc}
}

func (h *TeamHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	var req model.CreateTeamRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	team, err := h.teamSvc.Create(r.Context(), userID, req)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, team)
}

func (h *TeamHandler) List(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	teams, err := h.teamSvc.ListByUser(r.Context(), userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	if teams == nil {
		teams = []model.Team{}
	}

	respondJSON(w, http.StatusOK, teams)
}

func (h *TeamHandler) Invite(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	teamIDStr := chi.URLParam(r, "id")
	teamID, err := strconv.ParseInt(teamIDStr, 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid team id")
		return
	}

	var req model.InviteRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.teamSvc.InviteUser(r.Context(), userID, teamID, req); err != nil {
		switch {
		case errors.Is(err, service.ErrNotMember):
			respondError(w, http.StatusForbidden, err.Error())
		case errors.Is(err, service.ErrNoPermission):
			respondError(w, http.StatusForbidden, err.Error())
		case errors.Is(err, service.ErrUserNotFound):
			respondError(w, http.StatusNotFound, err.Error())
		case errors.Is(err, service.ErrAlreadyMember):
			respondError(w, http.StatusConflict, err.Error())
		default:
			respondError(w, http.StatusBadRequest, err.Error())
		}
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "user invited successfully"})
}
