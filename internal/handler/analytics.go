package handler

import (
	"net/http"

	"taskmanager/internal/repository"
)

type AnalyticsHandler struct {
	analyticsRepo repository.AnalyticsRepository
}

func NewAnalyticsHandler(analyticsRepo repository.AnalyticsRepository) *AnalyticsHandler {
	return &AnalyticsHandler{analyticsRepo: analyticsRepo}
}

func (h *AnalyticsHandler) GetTeamStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.analyticsRepo.GetTeamStats(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	respondJSON(w, http.StatusOK, stats)
}

func (h *AnalyticsHandler) GetTopCreators(w http.ResponseWriter, r *http.Request) {
	creators, err := h.analyticsRepo.GetTopCreators(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	respondJSON(w, http.StatusOK, creators)
}

func (h *AnalyticsHandler) GetOrphanedTasks(w http.ResponseWriter, r *http.Request) {
	tasks, err := h.analyticsRepo.GetOrphanedTasks(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	respondJSON(w, http.StatusOK, tasks)
}
