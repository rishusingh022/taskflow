package service

import (
	"context"
	"time"

	"taskflow/internal/model"

	"github.com/google/uuid"
)

type TaskService struct {
	taskRepo    TaskRepository
	projectRepo ProjectRepository
}

func NewTaskService(taskRepo TaskRepository, projectRepo ProjectRepository) *TaskService {
	return &TaskService{
		taskRepo:    taskRepo,
		projectRepo: projectRepo,
	}
}

func (s *TaskService) ListByProject(ctx context.Context, userID, projectID uuid.UUID, filter model.TaskFilter) ([]model.Task, int, error) {
	proj, err := s.projectRepo.FindByID(ctx, projectID)
	if err != nil {
		return nil, 0, err
	}
	if proj == nil {
		return nil, 0, model.ErrNotFound
	}
	ok, err := s.projectRepo.UserHasProjectAccess(ctx, userID, projectID)
	if err != nil {
		return nil, 0, err
	}
	if !ok {
		return nil, 0, model.ErrForbidden
	}

	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.Limit < 1 || filter.Limit > 100 {
		filter.Limit = 20
	}

	return s.taskRepo.FindByProject(ctx, projectID, filter)
}

func (s *TaskService) Create(ctx context.Context, projectID, userID uuid.UUID, req model.CreateTaskRequest) (*model.Task, error) {
	proj, err := s.projectRepo.FindByID(ctx, projectID)
	if err != nil {
		return nil, err
	}
	if proj == nil {
		return nil, model.ErrNotFound
	}
	if proj.OwnerID != userID {
		return nil, model.ErrForbidden
	}

	var dueDate *time.Time
	if req.DueDate != nil {
		parsed, err := time.Parse("2006-01-02", *req.DueDate)
		if err != nil {
			return nil, model.ErrInvalidInput
		}
		dueDate = &parsed
	}

	task := &model.Task{
		Title:       req.Title,
		Description: req.Description,
		Status:      model.StatusTodo,
		Priority:    req.Priority,
		ProjectID:   projectID,
		AssigneeID:  req.AssigneeID,
		DueDate:     dueDate,
		CreatedBy:   userID,
	}

	if err := s.taskRepo.Create(ctx, task); err != nil {
		return nil, err
	}
	return task, nil
}

func (s *TaskService) Update(ctx context.Context, taskID, userID uuid.UUID, req model.UpdateTaskRequest) (*model.Task, error) {
	task, err := s.taskRepo.FindByID(ctx, taskID)
	if err != nil {
		return nil, err
	}
	if task == nil {
		return nil, model.ErrNotFound
	}

	proj, err := s.projectRepo.FindByID(ctx, task.ProjectID)
	if err != nil {
		return nil, err
	}
	if proj == nil {
		return nil, model.ErrNotFound
	}
	allowed := proj.OwnerID == userID
	if !allowed && task.AssigneeID != nil && *task.AssigneeID == userID {
		allowed = true
	}
	if !allowed {
		return nil, model.ErrForbidden
	}

	// apply partial updates
	if req.Title != nil {
		task.Title = *req.Title
	}
	if req.Description != nil {
		task.Description = req.Description
	}
	if req.Status != nil {
		task.Status = *req.Status
	}
	if req.Priority != nil {
		task.Priority = *req.Priority
	}
	if req.AssigneeID != nil {
		task.AssigneeID = req.AssigneeID
	}
	if req.DueDate != nil {
		parsed, err := time.Parse("2006-01-02", *req.DueDate)
		if err != nil {
			return nil, model.ErrInvalidInput
		}
		task.DueDate = &parsed
	}

	if err := s.taskRepo.Update(ctx, task); err != nil {
		return nil, err
	}
	return task, nil
}

func (s *TaskService) Delete(ctx context.Context, taskID, userID uuid.UUID) error {
	task, err := s.taskRepo.FindByID(ctx, taskID)
	if err != nil {
		return err
	}
	if task == nil {
		return model.ErrNotFound
	}

	proj, err := s.projectRepo.FindByID(ctx, task.ProjectID)
	if err != nil {
		return err
	}
	if proj == nil {
		return model.ErrNotFound
	}
	if proj.OwnerID != userID {
		return model.ErrForbidden
	}

	return s.taskRepo.Delete(ctx, taskID)
}
