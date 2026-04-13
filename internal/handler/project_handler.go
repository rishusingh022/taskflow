package handler

import (
	"errors"
	"net/http"
	"strconv"

	"taskflow/internal/middleware"
	"taskflow/internal/model"
	"taskflow/internal/service"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type ProjectHandler struct {
	projectSvc *service.ProjectService
}

func NewProjectHandler(projectSvc *service.ProjectService) *ProjectHandler {
	return &ProjectHandler{projectSvc: projectSvc}
}

func (h *ProjectHandler) List(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}

	projects, total, err := h.projectSvc.List(r.Context(), userID, page, limit)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	if projects == nil {
		projects = []model.Project{}
	}

	respondJSON(w, http.StatusOK, newPaginatedResponse(projects, total, page, limit))
}

func (h *ProjectHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())

	var req model.CreateProjectRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if errs := req.Validate(); len(errs) > 0 {
		respondValidationError(w, errs)
		return
	}

	proj, err := h.projectSvc.Create(r.Context(), req, userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	respondJSON(w, http.StatusCreated, proj)
}

func (h *ProjectHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	projectID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid project id")
		return
	}

	proj, err := h.projectSvc.GetByID(r.Context(), projectID)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, proj)
}

func (h *ProjectHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	projectID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid project id")
		return
	}

	var req model.UpdateProjectRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	proj, err := h.projectSvc.Update(r.Context(), projectID, userID, req)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, proj)
}

func (h *ProjectHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	projectID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid project id")
		return
	}

	if err := h.projectSvc.Delete(r.Context(), projectID, userID); err != nil {
		handleServiceError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *ProjectHandler) GetStats(w http.ResponseWriter, r *http.Request) {
	projectID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid project id")
		return
	}

	stats, err := h.projectSvc.GetStats(r.Context(), projectID)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, stats)
}

func handleServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, model.ErrNotFound):
		respondError(w, http.StatusNotFound, "not found")
	case errors.Is(err, model.ErrForbidden):
		respondError(w, http.StatusForbidden, "forbidden")
	case errors.Is(err, model.ErrAlreadyExists):
		respondError(w, http.StatusConflict, "already exists")
	case errors.Is(err, model.ErrInvalidInput):
		respondError(w, http.StatusBadRequest, "invalid input")
	default:
		respondError(w, http.StatusInternalServerError, "internal server error")
	}
}
