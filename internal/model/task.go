package model

import (
	"time"

	"github.com/google/uuid"
)

type TaskStatus string

const (
	StatusTodo       TaskStatus = "todo"
	StatusInProgress TaskStatus = "in_progress"
	StatusDone       TaskStatus = "done"
)

func (s TaskStatus) Valid() bool {
	switch s {
	case StatusTodo, StatusInProgress, StatusDone:
		return true
	}
	return false
}

type TaskPriority string

const (
	PriorityLow    TaskPriority = "low"
	PriorityMedium TaskPriority = "medium"
	PriorityHigh   TaskPriority = "high"
)

func (p TaskPriority) Valid() bool {
	switch p {
	case PriorityLow, PriorityMedium, PriorityHigh:
		return true
	}
	return false
}

type Task struct {
	ID          uuid.UUID    `json:"id" db:"id"`
	Title       string       `json:"title" db:"title"`
	Description *string      `json:"description,omitempty" db:"description"`
	Status      TaskStatus   `json:"status" db:"status"`
	Priority    TaskPriority `json:"priority" db:"priority"`
	ProjectID   uuid.UUID    `json:"project_id" db:"project_id"`
	AssigneeID  *uuid.UUID   `json:"assignee_id,omitempty" db:"assignee_id"`
	DueDate     *time.Time   `json:"due_date,omitempty" db:"due_date"`
	CreatedAt   time.Time    `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at" db:"updated_at"`
	CreatedBy   uuid.UUID    `json:"created_by" db:"created_by"`
}

type CreateTaskRequest struct {
	Title       string       `json:"title"`
	Description *string      `json:"description,omitempty"`
	Priority    TaskPriority `json:"priority"`
	AssigneeID  *uuid.UUID   `json:"assignee_id,omitempty"`
	DueDate     *string      `json:"due_date,omitempty"`
}

func (r *CreateTaskRequest) Validate() map[string]string {
	errs := make(map[string]string)
	if r.Title == "" {
		errs["title"] = "is required"
	}
	if r.Priority == "" {
		r.Priority = PriorityMedium
	} else if !r.Priority.Valid() {
		errs["priority"] = "must be low, medium, or high"
	}
	return errs
}

type UpdateTaskRequest struct {
	Title       *string       `json:"title,omitempty"`
	Description *string       `json:"description,omitempty"`
	Status      *TaskStatus   `json:"status,omitempty"`
	Priority    *TaskPriority `json:"priority,omitempty"`
	AssigneeID  *uuid.UUID    `json:"assignee_id,omitempty"`
	DueDate     *string       `json:"due_date,omitempty"`
}

func (r *UpdateTaskRequest) Validate() map[string]string {
	errs := make(map[string]string)
	if r.Status != nil && !r.Status.Valid() {
		errs["status"] = "must be todo, in_progress, or done"
	}
	if r.Priority != nil && !r.Priority.Valid() {
		errs["priority"] = "must be low, medium, or high"
	}
	return errs
}

type TaskFilter struct {
	Status   *TaskStatus
	Assignee *uuid.UUID
	Page     int
	Limit    int
}
