package handler

import (
	"net/http"
	"strconv"

	"taskflow/internal/middleware"
	"taskflow/internal/model"
	"taskflow/internal/service"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type TaskHandler struct {
	taskSvc *service.TaskService
}

func NewTaskHandler(taskSvc *service.TaskService) *TaskHandler {
	return &TaskHandler{taskSvc: taskSvc}
}

func (h *TaskHandler) ListByProject(w http.ResponseWriter, r *http.Request) {
	projectID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid project id")
		return
	}

	filter := model.TaskFilter{}
	if s := r.URL.Query().Get("status"); s != "" {
		status := model.TaskStatus(s)
		if !status.Valid() {
			respondError(w, http.StatusBadRequest, "invalid status filter")
			return
		}
		filter.Status = &status
	}
	if a := r.URL.Query().Get("assignee"); a != "" {
		assigneeID, err := uuid.Parse(a)
		if err != nil {
			respondError(w, http.StatusBadRequest, "invalid assignee filter")
			return
		}
		filter.Assignee = &assigneeID
	}

	filter.Page, _ = strconv.Atoi(r.URL.Query().Get("page"))
	filter.Limit, _ = strconv.Atoi(r.URL.Query().Get("limit"))
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.Limit < 1 {
		filter.Limit = 20
	}

	tasks, total, err := h.taskSvc.ListByProject(r.Context(), projectID, filter)
	if err != nil {
		handleServiceError(w, err)
		return
	}
	if tasks == nil {
		tasks = []model.Task{}
	}

	respondJSON(w, http.StatusOK, newPaginatedResponse(tasks, total, filter.Page, filter.Limit))
}

func (h *TaskHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	projectID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid project id")
		return
	}

	var req model.CreateTaskRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if errs := req.Validate(); len(errs) > 0 {
		respondValidationError(w, errs)
		return
	}

	task, err := h.taskSvc.Create(r.Context(), projectID, userID, req)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	respondJSON(w, http.StatusCreated, task)
}

func (h *TaskHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	taskID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid task id")
		return
	}
	_ = userID // we don't restrict who can update, only delete is restricted

	var req model.UpdateTaskRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if errs := req.Validate(); len(errs) > 0 {
		respondValidationError(w, errs)
		return
	}

	task, err := h.taskSvc.Update(r.Context(), taskID, userID, req)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, task)
}

func (h *TaskHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	taskID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid task id")
		return
	}

	if err := h.taskSvc.Delete(r.Context(), taskID, userID); err != nil {
		handleServiceError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
