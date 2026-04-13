package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"taskflow/internal/model"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type TaskRepo struct {
	db *sqlx.DB
}

func NewTaskRepo(db *sqlx.DB) *TaskRepo {
	return &TaskRepo{db: db}
}

func (r *TaskRepo) FindByProject(ctx context.Context, projectID uuid.UUID, filter model.TaskFilter) ([]model.Task, int, error) {
	var conditions []string
	var args []interface{}
	argIdx := 1

	conditions = append(conditions, fmt.Sprintf("project_id = $%d", argIdx))
	args = append(args, projectID)
	argIdx++

	if filter.Status != nil {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, *filter.Status)
		argIdx++
	}
	if filter.Assignee != nil {
		conditions = append(conditions, fmt.Sprintf("assignee_id = $%d", argIdx))
		args = append(args, *filter.Assignee)
		argIdx++
	}

	where := strings.Join(conditions, " AND ")

	// total for pagination
	var total int
	countQ := fmt.Sprintf(`SELECT COUNT(*) FROM tasks WHERE %s`, where)
	if err := r.db.GetContext(ctx, &total, countQ, args...); err != nil {
		return nil, 0, err
	}

	query := fmt.Sprintf(`
		SELECT * FROM tasks WHERE %s
		ORDER BY
			CASE priority WHEN 'high' THEN 1 WHEN 'medium' THEN 2 WHEN 'low' THEN 3 END,
			created_at DESC
		LIMIT $%d OFFSET $%d`, where, argIdx, argIdx+1)

	args = append(args, filter.Limit, (filter.Page-1)*filter.Limit)

	var tasks []model.Task
	err := r.db.SelectContext(ctx, &tasks, query, args...)
	return tasks, total, err
}

func (r *TaskRepo) FindByID(ctx context.Context, id uuid.UUID) (*model.Task, error) {
	var task model.Task
	err := r.db.GetContext(ctx, &task, `SELECT * FROM tasks WHERE id = $1`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &task, err
}

func (r *TaskRepo) Create(ctx context.Context, task *model.Task) error {
	task.ID = uuid.New()
	query := `
		INSERT INTO tasks (id, title, description, status, priority, project_id, assignee_id, due_date, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING created_at, updated_at`
	return r.db.QueryRowContext(ctx, query,
		task.ID, task.Title, task.Description, task.Status, task.Priority,
		task.ProjectID, task.AssigneeID, task.DueDate, task.CreatedBy,
	).Scan(&task.CreatedAt, &task.UpdatedAt)
}

func (r *TaskRepo) Update(ctx context.Context, task *model.Task) error {
	query := `
		UPDATE tasks SET
			title = $1, description = $2, status = $3, priority = $4,
			assignee_id = $5, due_date = $6, updated_at = NOW()
		WHERE id = $7
		RETURNING updated_at`
	return r.db.QueryRowContext(ctx, query,
		task.Title, task.Description, task.Status, task.Priority,
		task.AssigneeID, task.DueDate, task.ID,
	).Scan(&task.UpdatedAt)
}

func (r *TaskRepo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM tasks WHERE id = $1`, id)
	return err
}
