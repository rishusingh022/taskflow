package service

import (
	"context"
	"testing"

	"taskflow/internal/model"

	"github.com/google/uuid"
)

// --- mock project repo ---

type mockProjectRepo struct {
	projects map[uuid.UUID]*model.Project
}

func newMockProjectRepo() *mockProjectRepo {
	return &mockProjectRepo{projects: make(map[uuid.UUID]*model.Project)}
}

func (m *mockProjectRepo) FindByUserAccess(ctx context.Context, userID uuid.UUID, page, limit int) ([]model.Project, int, error) {
	var result []model.Project
	for _, p := range m.projects {
		if p.OwnerID == userID {
			result = append(result, *p)
		}
	}
	return result, len(result), nil
}

func (m *mockProjectRepo) FindByID(ctx context.Context, id uuid.UUID) (*model.Project, error) {
	p, ok := m.projects[id]
	if !ok {
		return nil, nil
	}
	return p, nil
}

func (m *mockProjectRepo) Create(ctx context.Context, proj *model.Project) error {
	proj.ID = uuid.New()
	m.projects[proj.ID] = proj
	return nil
}

func (m *mockProjectRepo) Update(ctx context.Context, proj *model.Project) error {
	m.projects[proj.ID] = proj
	return nil
}

func (m *mockProjectRepo) Delete(ctx context.Context, id uuid.UUID) error {
	delete(m.projects, id)
	return nil
}

func (m *mockProjectRepo) GetStats(ctx context.Context, projectID uuid.UUID) (*model.ProjectStats, error) {
	if _, ok := m.projects[projectID]; !ok {
		return nil, nil
	}
	return &model.ProjectStats{
		Total:    3,
		ByStatus: map[string]int{"todo": 1, "in_progress": 1, "done": 1},
	}, nil
}

// --- mock task repo (minimal, used by project service) ---

type mockTaskRepo struct {
	tasks map[uuid.UUID]*model.Task
}

func newMockTaskRepo() *mockTaskRepo {
	return &mockTaskRepo{tasks: make(map[uuid.UUID]*model.Task)}
}

func (m *mockTaskRepo) FindByProject(ctx context.Context, projectID uuid.UUID, filter model.TaskFilter) ([]model.Task, int, error) {
	var result []model.Task
	for _, t := range m.tasks {
		if t.ProjectID == projectID {
			result = append(result, *t)
		}
	}
	return result, len(result), nil
}

func (m *mockTaskRepo) FindByID(ctx context.Context, id uuid.UUID) (*model.Task, error) {
	t, ok := m.tasks[id]
	if !ok {
		return nil, nil
	}
	return t, nil
}

func (m *mockTaskRepo) Create(ctx context.Context, task *model.Task) error {
	task.ID = uuid.New()
	m.tasks[task.ID] = task
	return nil
}

func (m *mockTaskRepo) Update(ctx context.Context, task *model.Task) error {
	m.tasks[task.ID] = task
	return nil
}

func (m *mockTaskRepo) Delete(ctx context.Context, id uuid.UUID) error {
	delete(m.tasks, id)
	return nil
}

// --- project service tests ---

func TestProjectService_Create(t *testing.T) {
	projRepo := newMockProjectRepo()
	taskRepo := newMockTaskRepo()
	svc := NewProjectService(projRepo, taskRepo)

	ownerID := uuid.New()
	proj, err := svc.Create(context.Background(), model.CreateProjectRequest{
		Name:        "Test Project",
		Description: strPtr("A test project"),
	}, ownerID)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if proj.Name != "Test Project" {
		t.Errorf("expected name Test Project, got %s", proj.Name)
	}
	if proj.OwnerID != ownerID {
		t.Error("expected owner to be set")
	}
}

func TestProjectService_UpdateByOwner(t *testing.T) {
	projRepo := newMockProjectRepo()
	taskRepo := newMockTaskRepo()
	svc := NewProjectService(projRepo, taskRepo)

	ownerID := uuid.New()
	proj, _ := svc.Create(context.Background(), model.CreateProjectRequest{Name: "Original"}, ownerID)

	updated, err := svc.Update(context.Background(), proj.ID, ownerID, model.UpdateProjectRequest{
		Name: strPtr("Updated"),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.Name != "Updated" {
		t.Errorf("expected Updated, got %s", updated.Name)
	}
}

func TestProjectService_UpdateByNonOwner(t *testing.T) {
	projRepo := newMockProjectRepo()
	taskRepo := newMockTaskRepo()
	svc := NewProjectService(projRepo, taskRepo)

	ownerID := uuid.New()
	otherID := uuid.New()
	proj, _ := svc.Create(context.Background(), model.CreateProjectRequest{Name: "Mine"}, ownerID)

	_, err := svc.Update(context.Background(), proj.ID, otherID, model.UpdateProjectRequest{
		Name: strPtr("Nope"),
	})
	if err != model.ErrForbidden {
		t.Errorf("expected ErrForbidden, got %v", err)
	}
}

func TestProjectService_DeleteByOwner(t *testing.T) {
	projRepo := newMockProjectRepo()
	taskRepo := newMockTaskRepo()
	svc := NewProjectService(projRepo, taskRepo)

	ownerID := uuid.New()
	proj, _ := svc.Create(context.Background(), model.CreateProjectRequest{Name: "To Delete"}, ownerID)

	err := svc.Delete(context.Background(), proj.ID, ownerID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// should be gone
	_, err = svc.GetByID(context.Background(), proj.ID)
	if err != model.ErrNotFound {
		t.Errorf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestProjectService_DeleteByNonOwner(t *testing.T) {
	projRepo := newMockProjectRepo()
	taskRepo := newMockTaskRepo()
	svc := NewProjectService(projRepo, taskRepo)

	ownerID := uuid.New()
	otherID := uuid.New()
	proj, _ := svc.Create(context.Background(), model.CreateProjectRequest{Name: "Protected"}, ownerID)

	err := svc.Delete(context.Background(), proj.ID, otherID)
	if err != model.ErrForbidden {
		t.Errorf("expected ErrForbidden, got %v", err)
	}
}

func TestProjectService_GetNotFound(t *testing.T) {
	projRepo := newMockProjectRepo()
	taskRepo := newMockTaskRepo()
	svc := NewProjectService(projRepo, taskRepo)

	_, err := svc.GetByID(context.Background(), uuid.New())
	if err != model.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestProjectService_GetStats(t *testing.T) {
	projRepo := newMockProjectRepo()
	taskRepo := newMockTaskRepo()
	svc := NewProjectService(projRepo, taskRepo)

	ownerID := uuid.New()
	proj, _ := svc.Create(context.Background(), model.CreateProjectRequest{Name: "Stats"}, ownerID)

	stats, err := svc.GetStats(context.Background(), proj.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stats.Total != 3 {
		t.Errorf("expected total 3 from mock, got %d", stats.Total)
	}
}

func TestProjectService_GetStatsNotFound(t *testing.T) {
	projRepo := newMockProjectRepo()
	taskRepo := newMockTaskRepo()
	svc := NewProjectService(projRepo, taskRepo)

	_, err := svc.GetStats(context.Background(), uuid.New())
	if err != model.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestProjectService_ListPagination(t *testing.T) {
	projRepo := newMockProjectRepo()
	taskRepo := newMockTaskRepo()
	svc := NewProjectService(projRepo, taskRepo)

	// invalid page/limit should be clamped
	projects, total, err := svc.List(context.Background(), uuid.New(), -1, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 0 {
		t.Errorf("expected 0 total, got %d", total)
	}
	if projects != nil && len(projects) > 0 {
		t.Error("expected empty projects")
	}
}

func strPtr(s string) *string { return &s }
