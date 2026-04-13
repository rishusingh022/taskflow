package model

import (
	"time"

	"github.com/google/uuid"
)

type Project struct {
	ID          uuid.UUID `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	Description *string   `json:"description,omitempty" db:"description"`
	OwnerID     uuid.UUID `json:"owner_id" db:"owner_id"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

// ProjectWithTasks is returned when fetching a single project
type ProjectWithTasks struct {
	Project
	Tasks []Task `json:"tasks"`
}

type CreateProjectRequest struct {
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
}

func (r *CreateProjectRequest) Validate() map[string]string {
	errs := make(map[string]string)
	if r.Name == "" {
		errs["name"] = "is required"
	}
	return errs
}

type UpdateProjectRequest struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
}

type ProjectStats struct {
	Total      int                  `json:"total"`
	ByStatus   map[string]int       `json:"by_status"`
	ByAssignee []AssigneeTaskCount  `json:"by_assignee"`
}

type AssigneeTaskCount struct {
	UserID *uuid.UUID `json:"user_id"`
	Name   string     `json:"name"`
	Count  int        `json:"count"`
}
