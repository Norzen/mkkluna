package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"taskmanager/internal/model"
)

type TaskRepository interface {
	Create(ctx context.Context, task *model.Task) (int64, error)
	GetByID(ctx context.Context, id int64) (*model.Task, error)
	Update(ctx context.Context, task *model.Task) error
	List(ctx context.Context, filter model.TaskFilter) ([]model.Task, int64, error)
	AddHistory(ctx context.Context, h *model.TaskHistory) error
	GetHistory(ctx context.Context, taskID int64) ([]model.TaskHistory, error)
	AddComment(ctx context.Context, c *model.TaskComment) (int64, error)
	GetComments(ctx context.Context, taskID int64) ([]model.TaskComment, error)
}

type taskRepo struct {
	db *sql.DB
}

func NewTaskRepository(db *sql.DB) TaskRepository {
	return &taskRepo{db: db}
}

func (r *taskRepo) Create(ctx context.Context, task *model.Task) (int64, error) {
	query := `INSERT INTO tasks (title, description, status, priority, team_id, assignee_id, created_by, due_date)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
	result, err := r.db.ExecContext(ctx, query,
		task.Title, task.Description, task.Status, task.Priority,
		task.TeamID, task.AssigneeID, task.CreatedBy, task.DueDate,
	)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (r *taskRepo) GetByID(ctx context.Context, id int64) (*model.Task, error) {
	query := `SELECT id, title, description, status, priority, team_id, assignee_id, created_by, due_date, created_at, updated_at
		FROM tasks WHERE id = ?`
	t := &model.Task{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&t.ID, &t.Title, &t.Description, &t.Status, &t.Priority,
		&t.TeamID, &t.AssigneeID, &t.CreatedBy, &t.DueDate, &t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return t, nil
}

func (r *taskRepo) Update(ctx context.Context, task *model.Task) error {
	query := `UPDATE tasks SET title=?, description=?, status=?, priority=?, assignee_id=?, due_date=?, updated_at=NOW()
		WHERE id=?`
	_, err := r.db.ExecContext(ctx, query,
		task.Title, task.Description, task.Status, task.Priority,
		task.AssigneeID, task.DueDate, task.ID,
	)
	return err
}

func (r *taskRepo) List(ctx context.Context, filter model.TaskFilter) ([]model.Task, int64, error) {
	var conditions []string
	var args []any

	if filter.TeamID > 0 {
		conditions = append(conditions, "team_id = ?")
		args = append(args, filter.TeamID)
	}
	if filter.Status != "" {
		conditions = append(conditions, "status = ?")
		args = append(args, filter.Status)
	}
	if filter.AssigneeID > 0 {
		conditions = append(conditions, "assignee_id = ?")
		args = append(args, filter.AssigneeID)
	}

	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + strings.Join(conditions, " AND ")
	}

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM tasks %s", where)
	var total int64
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PageSize < 1 || filter.PageSize > 100 {
		filter.PageSize = 20
	}
	offset := (filter.Page - 1) * filter.PageSize

	dataQuery := fmt.Sprintf(`SELECT id, title, description, status, priority, team_id, assignee_id, created_by, due_date, created_at, updated_at
		FROM tasks %s ORDER BY created_at DESC LIMIT ? OFFSET ?`, where)
	args = append(args, filter.PageSize, offset)

	rows, err := r.db.QueryContext(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var tasks []model.Task
	for rows.Next() {
		var t model.Task
		if err := rows.Scan(
			&t.ID, &t.Title, &t.Description, &t.Status, &t.Priority,
			&t.TeamID, &t.AssigneeID, &t.CreatedBy, &t.DueDate, &t.CreatedAt, &t.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		tasks = append(tasks, t)
	}
	return tasks, total, rows.Err()
}

func (r *taskRepo) AddHistory(ctx context.Context, h *model.TaskHistory) error {
	query := `INSERT INTO task_history (task_id, changed_by, field_name, old_value, new_value) VALUES (?, ?, ?, ?, ?)`
	_, err := r.db.ExecContext(ctx, query, h.TaskID, h.ChangedBy, h.FieldName, h.OldValue, h.NewValue)
	return err
}

func (r *taskRepo) GetHistory(ctx context.Context, taskID int64) ([]model.TaskHistory, error) {
	query := `SELECT id, task_id, changed_by, field_name, old_value, new_value, changed_at
		FROM task_history WHERE task_id = ? ORDER BY changed_at DESC`
	rows, err := r.db.QueryContext(ctx, query, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var history []model.TaskHistory
	for rows.Next() {
		var h model.TaskHistory
		if err := rows.Scan(&h.ID, &h.TaskID, &h.ChangedBy, &h.FieldName, &h.OldValue, &h.NewValue, &h.ChangedAt); err != nil {
			return nil, err
		}
		history = append(history, h)
	}
	return history, rows.Err()
}

func (r *taskRepo) AddComment(ctx context.Context, c *model.TaskComment) (int64, error) {
	query := `INSERT INTO task_comments (task_id, user_id, content) VALUES (?, ?, ?)`
	result, err := r.db.ExecContext(ctx, query, c.TaskID, c.UserID, c.Content)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (r *taskRepo) GetComments(ctx context.Context, taskID int64) ([]model.TaskComment, error) {
	query := `SELECT id, task_id, user_id, content, created_at, updated_at
		FROM task_comments WHERE task_id = ? ORDER BY created_at ASC`
	rows, err := r.db.QueryContext(ctx, query, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var comments []model.TaskComment
	for rows.Next() {
		var c model.TaskComment
		if err := rows.Scan(&c.ID, &c.TaskID, &c.UserID, &c.Content, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		comments = append(comments, c)
	}
	return comments, rows.Err()
}
