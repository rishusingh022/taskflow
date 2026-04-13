package model_test

import (
	"testing"

	"taskflow/internal/model"
)

func TestRegisterRequest_Validate(t *testing.T) {
	tests := []struct {
		name     string
		req      model.RegisterRequest
		wantErrs int
	}{
		{
			name:     "valid request",
			req:      model.RegisterRequest{Name: "Test User", Email: "test@test.com", Password: "password123"},
			wantErrs: 0,
		},
		{
			name:     "empty fields",
			req:      model.RegisterRequest{},
			wantErrs: 3,
		},
		{
			name:     "short password",
			req:      model.RegisterRequest{Name: "Test", Email: "t@t.com", Password: "abc"},
			wantErrs: 1,
		},
		{
			name:     "missing name only",
			req:      model.RegisterRequest{Email: "t@t.com", Password: "password123"},
			wantErrs: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := tt.req.Validate()
			if len(errs) != tt.wantErrs {
				t.Errorf("expected %d errors, got %d: %v", tt.wantErrs, len(errs), errs)
			}
		})
	}
}

func TestLoginRequest_Validate(t *testing.T) {
	tests := []struct {
		name     string
		req      model.LoginRequest
		wantErrs int
	}{
		{
			name:     "valid",
			req:      model.LoginRequest{Email: "test@test.com", Password: "password123"},
			wantErrs: 0,
		},
		{
			name:     "all empty",
			req:      model.LoginRequest{},
			wantErrs: 2,
		},
		{
			name:     "no password",
			req:      model.LoginRequest{Email: "test@test.com"},
			wantErrs: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := tt.req.Validate()
			if len(errs) != tt.wantErrs {
				t.Errorf("expected %d errors, got %d: %v", tt.wantErrs, len(errs), errs)
			}
		})
	}
}

func TestTaskStatus_Valid(t *testing.T) {
	valid := []model.TaskStatus{model.StatusTodo, model.StatusInProgress, model.StatusDone}
	for _, s := range valid {
		if !s.Valid() {
			t.Errorf("expected %s to be valid", s)
		}
	}

	if model.TaskStatus("invalid").Valid() {
		t.Error("expected 'invalid' to be invalid")
	}
	if model.TaskStatus("").Valid() {
		t.Error("expected empty to be invalid")
	}
}

func TestTaskPriority_Valid(t *testing.T) {
	valid := []model.TaskPriority{model.PriorityLow, model.PriorityMedium, model.PriorityHigh}
	for _, p := range valid {
		if !p.Valid() {
			t.Errorf("expected %s to be valid", p)
		}
	}

	if model.TaskPriority("urgent").Valid() {
		t.Error("expected 'urgent' to be invalid")
	}
}

func TestCreateTaskRequest_Validate(t *testing.T) {
	tests := []struct {
		name     string
		req      model.CreateTaskRequest
		wantErrs int
	}{
		{
			name:     "valid with priority",
			req:      model.CreateTaskRequest{Title: "Test task", Priority: model.PriorityHigh},
			wantErrs: 0,
		},
		{
			name:     "valid without priority - defaults to medium",
			req:      model.CreateTaskRequest{Title: "Test task"},
			wantErrs: 0,
		},
		{
			name:     "empty title",
			req:      model.CreateTaskRequest{Priority: model.PriorityLow},
			wantErrs: 1,
		},
		{
			name:     "invalid priority",
			req:      model.CreateTaskRequest{Title: "t", Priority: "urgent"},
			wantErrs: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := tt.req.Validate()
			if len(errs) != tt.wantErrs {
				t.Errorf("expected %d errors, got %d: %v", tt.wantErrs, len(errs), errs)
			}
		})
	}
}

func TestUpdateTaskRequest_Validate(t *testing.T) {
	invalidStatus := model.TaskStatus("nope")
	invalidPriority := model.TaskPriority("critical")
	validStatus := model.StatusDone

	tests := []struct {
		name     string
		req      model.UpdateTaskRequest
		wantErrs int
	}{
		{
			name:     "empty update is valid",
			req:      model.UpdateTaskRequest{},
			wantErrs: 0,
		},
		{
			name:     "valid status",
			req:      model.UpdateTaskRequest{Status: &validStatus},
			wantErrs: 0,
		},
		{
			name:     "invalid status",
			req:      model.UpdateTaskRequest{Status: &invalidStatus},
			wantErrs: 1,
		},
		{
			name:     "invalid priority",
			req:      model.UpdateTaskRequest{Priority: &invalidPriority},
			wantErrs: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := tt.req.Validate()
			if len(errs) != tt.wantErrs {
				t.Errorf("expected %d errors, got %d: %v", tt.wantErrs, len(errs), errs)
			}
		})
	}
}

func TestCreateProjectRequest_Validate(t *testing.T) {
	tests := []struct {
		name     string
		req      model.CreateProjectRequest
		wantErrs int
	}{
		{
			name:     "valid",
			req:      model.CreateProjectRequest{Name: "My Project"},
			wantErrs: 0,
		},
		{
			name:     "empty name",
			req:      model.CreateProjectRequest{},
			wantErrs: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := tt.req.Validate()
			if len(errs) != tt.wantErrs {
				t.Errorf("expected %d errors, got %d: %v", tt.wantErrs, len(errs), errs)
			}
		})
	}
}
