package service

import (
	"context"
	"testing"

	"taskflow/internal/model"

	"github.com/google/uuid"
)

func TestTaskService_Create(t *testing.T) {
	projRepo := newMockProjectRepo()
	taskRepo := newMockTaskRepo()
	svc := NewTaskService(taskRepo, projRepo)

	ownerID := uuid.New()
	proj := &model.Project{ID: uuid.New(), Name: "Test", OwnerID: ownerID}
	projRepo.projects[proj.ID] = proj

	task, err := svc.Create(context.Background(), proj.ID, ownerID, model.CreateTaskRequest{
		Title:    "Test Task",
		Priority: model.PriorityHigh,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if task.Title != "Test Task" {
		t.Errorf("expected title Test Task, got %s", task.Title)
	}
	if task.Status != model.StatusTodo {
		t.Errorf("expected default status todo, got %s", string(task.Status))
	}
	if task.CreatedBy != ownerID {
		t.Error("expected created_by to be set")
	}
}

func TestTaskService_CreateWithDueDate(t *testing.T) {
	projRepo := newMockProjectRepo()
	taskRepo := newMockTaskRepo()
	svc := NewTaskService(taskRepo, projRepo)

	proj := &model.Project{ID: uuid.New(), OwnerID: uuid.New()}
	projRepo.projects[proj.ID] = proj

	dueDate := "2025-06-15"
	task, err := svc.Create(context.Background(), proj.ID, proj.OwnerID, model.CreateTaskRequest{
		Title:    "Due Task",
		Priority: model.PriorityMedium,
		DueDate:  &dueDate,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if task.DueDate == nil {
		t.Fatal("expected due date to be set")
	}
	if task.DueDate.Format("2006-01-02") != "2025-06-15" {
		t.Errorf("expected 2025-06-15, got %s", task.DueDate.Format("2006-01-02"))
	}
}

func TestTaskService_CreateInvalidDueDate(t *testing.T) {
	projRepo := newMockProjectRepo()
	taskRepo := newMockTaskRepo()
	svc := NewTaskService(taskRepo, projRepo)

	proj := &model.Project{ID: uuid.New(), OwnerID: uuid.New()}
	projRepo.projects[proj.ID] = proj

	badDate := "not-a-date"
	_, err := svc.Create(context.Background(), proj.ID, proj.OwnerID, model.CreateTaskRequest{
		Title:    "Bad Date",
		Priority: model.PriorityLow,
		DueDate:  &badDate,
	})
	if err != model.ErrInvalidInput {
		t.Errorf("expected ErrInvalidInput, got %v", err)
	}
}

func TestTaskService_CreateProjectNotFound(t *testing.T) {
	projRepo := newMockProjectRepo()
	taskRepo := newMockTaskRepo()
	svc := NewTaskService(taskRepo, projRepo)

	_, err := svc.Create(context.Background(), uuid.New(), uuid.New(), model.CreateTaskRequest{
		Title:    "Orphan",
		Priority: model.PriorityLow,
	})
	if err != model.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestTaskService_Update(t *testing.T) {
	projRepo := newMockProjectRepo()
	taskRepo := newMockTaskRepo()
	svc := NewTaskService(taskRepo, projRepo)

	taskID := uuid.New()
	status := model.StatusInProgress
	taskRepo.tasks[taskID] = &model.Task{
		ID:        taskID,
		Title:     "Original",
		Status:    model.StatusTodo,
		Priority:  model.PriorityMedium,
		ProjectID: uuid.New(),
	}

	newTitle := "Updated Title"
	updated, err := svc.Update(context.Background(), taskID, uuid.New(), model.UpdateTaskRequest{
		Title:  &newTitle,
		Status: &status,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.Title != "Updated Title" {
		t.Errorf("expected Updated Title, got %s", updated.Title)
	}
	if updated.Status != model.StatusInProgress {
		t.Errorf("expected in_progress, got %s", string(updated.Status))
	}
}

func TestTaskService_UpdateNotFound(t *testing.T) {
	projRepo := newMockProjectRepo()
	taskRepo := newMockTaskRepo()
	svc := NewTaskService(taskRepo, projRepo)

	newTitle := "Nope"
	_, err := svc.Update(context.Background(), uuid.New(), uuid.New(), model.UpdateTaskRequest{
		Title: &newTitle,
	})
	if err != model.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestTaskService_DeleteByProjectOwner(t *testing.T) {
	projRepo := newMockProjectRepo()
	taskRepo := newMockTaskRepo()
	svc := NewTaskService(taskRepo, projRepo)

	ownerID := uuid.New()
	creatorID := uuid.New()
	projID := uuid.New()
	taskID := uuid.New()

	projRepo.projects[projID] = &model.Project{ID: projID, OwnerID: ownerID}
	taskRepo.tasks[taskID] = &model.Task{
		ID:        taskID,
		ProjectID: projID,
		CreatedBy: creatorID,
	}

	err := svc.Delete(context.Background(), taskID, ownerID)
	if err != nil {
		t.Fatalf("project owner should be able to delete: %v", err)
	}
}

func TestTaskService_DeleteByTaskCreator(t *testing.T) {
	projRepo := newMockProjectRepo()
	taskRepo := newMockTaskRepo()
	svc := NewTaskService(taskRepo, projRepo)

	ownerID := uuid.New()
	creatorID := uuid.New()
	projID := uuid.New()
	taskID := uuid.New()

	projRepo.projects[projID] = &model.Project{ID: projID, OwnerID: ownerID}
	taskRepo.tasks[taskID] = &model.Task{
		ID:        taskID,
		ProjectID: projID,
		CreatedBy: creatorID,
	}

	err := svc.Delete(context.Background(), taskID, creatorID)
	if err != nil {
		t.Fatalf("task creator should be able to delete: %v", err)
	}
}

func TestTaskService_DeleteForbidden(t *testing.T) {
	projRepo := newMockProjectRepo()
	taskRepo := newMockTaskRepo()
	svc := NewTaskService(taskRepo, projRepo)

	ownerID := uuid.New()
	creatorID := uuid.New()
	randomID := uuid.New()
	projID := uuid.New()
	taskID := uuid.New()

	projRepo.projects[projID] = &model.Project{ID: projID, OwnerID: ownerID}
	taskRepo.tasks[taskID] = &model.Task{
		ID:        taskID,
		ProjectID: projID,
		CreatedBy: creatorID,
	}

	err := svc.Delete(context.Background(), taskID, randomID)
	if err != model.ErrForbidden {
		t.Errorf("expected ErrForbidden, got %v", err)
	}
}

func TestTaskService_DeleteNotFound(t *testing.T) {
	projRepo := newMockProjectRepo()
	taskRepo := newMockTaskRepo()
	svc := NewTaskService(taskRepo, projRepo)

	err := svc.Delete(context.Background(), uuid.New(), uuid.New())
	if err != model.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestTaskService_ListByProjectNotFound(t *testing.T) {
	projRepo := newMockProjectRepo()
	taskRepo := newMockTaskRepo()
	svc := NewTaskService(taskRepo, projRepo)

	_, _, err := svc.ListByProject(context.Background(), uuid.New(), model.TaskFilter{})
	if err != model.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestTaskService_ListByProjectPagination(t *testing.T) {
	projRepo := newMockProjectRepo()
	taskRepo := newMockTaskRepo()
	svc := NewTaskService(taskRepo, projRepo)

	projID := uuid.New()
	projRepo.projects[projID] = &model.Project{ID: projID, OwnerID: uuid.New()}

	// bounds clamping: page < 1, limit < 1
	tasks, total, err := svc.ListByProject(context.Background(), projID, model.TaskFilter{Page: -1, Limit: 0})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_ = tasks
	_ = total
}
