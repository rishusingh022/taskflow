package service

import (
	"context"

	"taskflow/internal/model"

	"github.com/google/uuid"
)

type ProjectService struct {
	projectRepo ProjectRepository
	taskRepo    TaskRepository
}

func NewProjectService(projectRepo ProjectRepository, taskRepo TaskRepository) *ProjectService {
	return &ProjectService{
		projectRepo: projectRepo,
		taskRepo:    taskRepo,
	}
}

func (s *ProjectService) List(ctx context.Context, userID uuid.UUID, page, limit int) ([]model.Project, int, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 50 {
		limit = 20
	}
	return s.projectRepo.FindByUserAccess(ctx, userID, page, limit)
}

func (s *ProjectService) GetByID(ctx context.Context, userID, projectID uuid.UUID) (*model.ProjectWithTasks, error) {
	proj, err := s.projectRepo.FindByID(ctx, projectID)
	if err != nil {
		return nil, err
	}
	if proj == nil {
		return nil, model.ErrNotFound
	}
	ok, err := s.projectRepo.UserHasProjectAccess(ctx, userID, projectID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, model.ErrForbidden
	}

	filter := model.TaskFilter{Page: 1, Limit: 100}
	tasks, _, err := s.taskRepo.FindByProject(ctx, projectID, filter)
	if err != nil {
		return nil, err
	}
	if tasks == nil {
		tasks = []model.Task{}
	}

	return &model.ProjectWithTasks{
		Project: *proj,
		Tasks:   tasks,
	}, nil
}

func (s *ProjectService) Create(ctx context.Context, req model.CreateProjectRequest, ownerID uuid.UUID) (*model.Project, error) {
	proj := &model.Project{
		Name:        req.Name,
		Description: req.Description,
		OwnerID:     ownerID,
	}
	if err := s.projectRepo.Create(ctx, proj); err != nil {
		return nil, err
	}
	return proj, nil
}

func (s *ProjectService) Update(ctx context.Context, projectID, userID uuid.UUID, req model.UpdateProjectRequest) (*model.Project, error) {
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

	if req.Name != nil {
		proj.Name = *req.Name
	}
	if req.Description != nil {
		proj.Description = req.Description
	}

	if err := s.projectRepo.Update(ctx, proj); err != nil {
		return nil, err
	}
	return proj, nil
}

func (s *ProjectService) Delete(ctx context.Context, projectID, userID uuid.UUID) error {
	proj, err := s.projectRepo.FindByID(ctx, projectID)
	if err != nil {
		return err
	}
	if proj == nil {
		return model.ErrNotFound
	}
	if proj.OwnerID != userID {
		return model.ErrForbidden
	}
	return s.projectRepo.Delete(ctx, projectID)
}

func (s *ProjectService) GetStats(ctx context.Context, userID, projectID uuid.UUID) (*model.ProjectStats, error) {
	proj, err := s.projectRepo.FindByID(ctx, projectID)
	if err != nil {
		return nil, err
	}
	if proj == nil {
		return nil, model.ErrNotFound
	}
	ok, err := s.projectRepo.UserHasProjectAccess(ctx, userID, projectID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, model.ErrForbidden
	}

	stats, err := s.projectRepo.GetStats(ctx, projectID)
	if err != nil {
		return nil, err
	}
	if stats == nil {
		return nil, model.ErrNotFound
	}
	return stats, nil
}
