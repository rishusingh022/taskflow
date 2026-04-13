package repository

import (
	"context"
	"database/sql"
	"errors"

	"taskflow/internal/model"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type ProjectRepo struct {
	db *sqlx.DB
}

func NewProjectRepo(db *sqlx.DB) *ProjectRepo {
	return &ProjectRepo{db: db}
}

func (r *ProjectRepo) FindByUserAccess(ctx context.Context, userID uuid.UUID, page, limit int) ([]model.Project, int, error) {
	// count total matching projects
	var total int
	countQ := `
		SELECT COUNT(DISTINCT p.id) FROM projects p
		LEFT JOIN tasks t ON t.project_id = p.id
		WHERE p.owner_id = $1 OR t.assignee_id = $1`
	if err := r.db.GetContext(ctx, &total, countQ, userID); err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * limit
	query := `
		SELECT DISTINCT p.id, p.name, p.description, p.owner_id, p.created_at
		FROM projects p
		LEFT JOIN tasks t ON t.project_id = p.id
		WHERE p.owner_id = $1 OR t.assignee_id = $1
		ORDER BY p.created_at DESC
		LIMIT $2 OFFSET $3`

	var projects []model.Project
	err := r.db.SelectContext(ctx, &projects, query, userID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	return projects, total, nil
}

func (r *ProjectRepo) FindByID(ctx context.Context, id uuid.UUID) (*model.Project, error) {
	var proj model.Project
	err := r.db.GetContext(ctx, &proj, `SELECT * FROM projects WHERE id = $1`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &proj, err
}

func (r *ProjectRepo) Create(ctx context.Context, proj *model.Project) error {
	proj.ID = uuid.New()
	query := `INSERT INTO projects (id, name, description, owner_id) VALUES ($1, $2, $3, $4) RETURNING created_at`
	return r.db.QueryRowContext(ctx, query, proj.ID, proj.Name, proj.Description, proj.OwnerID).Scan(&proj.CreatedAt)
}

func (r *ProjectRepo) Update(ctx context.Context, proj *model.Project) error {
	query := `UPDATE projects SET name = $1, description = $2 WHERE id = $3`
	_, err := r.db.ExecContext(ctx, query, proj.Name, proj.Description, proj.ID)
	return err
}

func (r *ProjectRepo) Delete(ctx context.Context, id uuid.UUID) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `DELETE FROM tasks WHERE project_id = $1`, id); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM projects WHERE id = $1`, id); err != nil {
		return err
	}
	return tx.Commit()
}

// GetStats returns per-project statistics
func (r *ProjectRepo) GetStats(ctx context.Context, projectID uuid.UUID) (*model.ProjectStats, error) {
	// verify project exists
	var exists bool
	if err := r.db.GetContext(ctx, &exists, `SELECT EXISTS(SELECT 1 FROM projects WHERE id = $1)`, projectID); err != nil {
		return nil, err
	}
	if !exists {
		return nil, nil
	}

	stats := &model.ProjectStats{
		ByStatus: make(map[string]int),
	}

	// total count
	if err := r.db.GetContext(ctx, &stats.Total, `SELECT COUNT(*) FROM tasks WHERE project_id = $1`, projectID); err != nil {
		return nil, err
	}

	// by status
	type statusRow struct {
		Status string `db:"status"`
		Count  int    `db:"count"`
	}
	var statusRows []statusRow
	if err := r.db.SelectContext(ctx, &statusRows,
		`SELECT status, COUNT(*) as count FROM tasks WHERE project_id = $1 GROUP BY status`, projectID); err != nil {
		return nil, err
	}

	for _, row := range statusRows {
		stats.ByStatus[row.Status] = row.Count
	}

	// fill in zeros for missing statuses
	for _, s := range []string{"todo", "in_progress", "done"} {
		if _, ok := stats.ByStatus[s]; !ok {
			stats.ByStatus[s] = 0
		}
	}

	// by assignee
	type assigneeRow struct {
		UserID *uuid.UUID `db:"user_id"`
		Name   *string    `db:"name"`
		Count  int        `db:"count"`
	}
	var assigneeRows []assigneeRow
	if err := r.db.SelectContext(ctx, &assigneeRows, `
		SELECT t.assignee_id as user_id, u.name, COUNT(*) as count
		FROM tasks t
		LEFT JOIN users u ON u.id = t.assignee_id
		WHERE t.project_id = $1
		GROUP BY t.assignee_id, u.name`, projectID); err != nil {
		return nil, err
	}

	for _, row := range assigneeRows {
		name := "Unassigned"
		if row.Name != nil {
			name = *row.Name
		}
		stats.ByAssignee = append(stats.ByAssignee, model.AssigneeTaskCount{
			UserID: row.UserID,
			Name:   name,
			Count:  row.Count,
		})
	}

	return stats, nil
}
