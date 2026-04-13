package service

import (
	"context"

	"taskflow/internal/model"

	"github.com/google/uuid"
)

// UserRepository defines the interface for user data access.
type UserRepository interface {
	Create(ctx context.Context, user *model.User) error
	FindByEmail(ctx context.Context, email string) (*model.User, error)
	FindByID(ctx context.Context, id uuid.UUID) (*model.User, error)
}

// ProjectRepository defines the interface for project data access.
type ProjectRepository interface {
	FindByUserAccess(ctx context.Context, userID uuid.UUID, page, limit int) ([]model.Project, int, error)
	FindByID(ctx context.Context, id uuid.UUID) (*model.Project, error)
	// UserHasProjectAccess is true if user owns the project or is assignee on at least one task in it.
	UserHasProjectAccess(ctx context.Context, userID, projectID uuid.UUID) (bool, error)
	Create(ctx context.Context, proj *model.Project) error
	Update(ctx context.Context, proj *model.Project) error
	Delete(ctx context.Context, id uuid.UUID) error
	GetStats(ctx context.Context, projectID uuid.UUID) (*model.ProjectStats, error)
}

// TaskRepository defines the interface for task data access.
type TaskRepository interface {
	FindByProject(ctx context.Context, projectID uuid.UUID, filter model.TaskFilter) ([]model.Task, int, error)
	FindByID(ctx context.Context, id uuid.UUID) (*model.Task, error)
	Create(ctx context.Context, task *model.Task) error
	Update(ctx context.Context, task *model.Task) error
	Delete(ctx context.Context, id uuid.UUID) error
}
