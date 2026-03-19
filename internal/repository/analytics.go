package repository

import (
	"context"
	"database/sql"

	"taskmanager/internal/model"
)

type AnalyticsRepository interface {
	// JOIN 3+ tables + aggregation:
	// team name, member count, done tasks in last 7 days
	GetTeamStats(ctx context.Context) ([]model.TeamStats, error)

	// Window function:
	// top-3 users by created tasks per team for the last month
	GetTopCreators(ctx context.Context) ([]model.TopCreator, error)

	// Subquery with related tables:
	// tasks where assignee is not a member of that task's team
	GetOrphanedTasks(ctx context.Context) ([]model.OrphanedTask, error)
}

type analyticsRepo struct {
	db *sql.DB
}

func NewAnalyticsRepository(db *sql.DB) AnalyticsRepository {
	return &analyticsRepo{db: db}
}

func (r *analyticsRepo) GetTeamStats(ctx context.Context) ([]model.TeamStats, error) {
	query := `
		SELECT
			t.id AS team_id,
			t.name AS team_name,
			COUNT(DISTINCT tm.user_id) AS member_count,
			COUNT(DISTINCT CASE
				WHEN tk.status = 'done' AND tk.updated_at >= DATE_SUB(NOW(), INTERVAL 7 DAY)
				THEN tk.id
			END) AS done_tasks_week
		FROM teams t
		LEFT JOIN team_members tm ON t.id = tm.team_id
		LEFT JOIN tasks tk ON t.id = tk.team_id
		GROUP BY t.id, t.name
		ORDER BY done_tasks_week DESC`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []model.TeamStats
	for rows.Next() {
		var s model.TeamStats
		if err := rows.Scan(&s.TeamID, &s.TeamName, &s.MemberCount, &s.DoneTasksWeek); err != nil {
			return nil, err
		}
		stats = append(stats, s)
	}
	return stats, rows.Err()
}

func (r *analyticsRepo) GetTopCreators(ctx context.Context) ([]model.TopCreator, error) {
	query := `
		SELECT user_id, user_name, team_id, team_name, task_count, rnk
		FROM (
			SELECT
				u.id AS user_id,
				u.name AS user_name,
				t.id AS team_id,
				t.name AS team_name,
				COUNT(tk.id) AS task_count,
				ROW_NUMBER() OVER (PARTITION BY t.id ORDER BY COUNT(tk.id) DESC) AS rnk
			FROM users u
			INNER JOIN team_members tm ON u.id = tm.user_id
			INNER JOIN teams t ON tm.team_id = t.id
			LEFT JOIN tasks tk ON tk.created_by = u.id
				AND tk.team_id = t.id
				AND tk.created_at >= DATE_SUB(NOW(), INTERVAL 30 DAY)
			GROUP BY u.id, u.name, t.id, t.name
		) ranked
		WHERE rnk <= 3
		ORDER BY team_id, rnk`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var creators []model.TopCreator
	for rows.Next() {
		var c model.TopCreator
		if err := rows.Scan(&c.UserID, &c.UserName, &c.TeamID, &c.TeamName, &c.TaskCount, &c.Rank); err != nil {
			return nil, err
		}
		creators = append(creators, c)
	}
	return creators, rows.Err()
}

func (r *analyticsRepo) GetOrphanedTasks(ctx context.Context) ([]model.OrphanedTask, error) {
	query := `
		SELECT
			tk.id AS task_id,
			tk.title AS task_title,
			tk.assignee_id,
			u.name AS assignee_name,
			tk.team_id,
			t.name AS team_name
		FROM tasks tk
		INNER JOIN users u ON tk.assignee_id = u.id
		INNER JOIN teams t ON tk.team_id = t.id
		WHERE tk.assignee_id IS NOT NULL
			AND NOT EXISTS (
				SELECT 1 FROM team_members tm
				WHERE tm.team_id = tk.team_id AND tm.user_id = tk.assignee_id
			)
		ORDER BY tk.id`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []model.OrphanedTask
	for rows.Next() {
		var o model.OrphanedTask
		if err := rows.Scan(&o.TaskID, &o.TaskTitle, &o.AssigneeID, &o.AssigneeName, &o.TeamID, &o.TeamName); err != nil {
			return nil, err
		}
		tasks = append(tasks, o)
	}
	return tasks, rows.Err()
}
