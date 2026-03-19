package repository

import (
	"context"
	"database/sql"

	"taskmanager/internal/model"
)

type TeamRepository interface {
	Create(ctx context.Context, team *model.Team) (int64, error)
	GetByID(ctx context.Context, id int64) (*model.Team, error)
	ListByUser(ctx context.Context, userID int64) ([]model.Team, error)
	AddMember(ctx context.Context, member *model.TeamMember) error
	GetMember(ctx context.Context, teamID, userID int64) (*model.TeamMember, error)
	GetMemberRole(ctx context.Context, teamID, userID int64) (model.TeamRole, error)
	IsMember(ctx context.Context, teamID, userID int64) (bool, error)
}

type teamRepo struct {
	db *sql.DB
}

func NewTeamRepository(db *sql.DB) TeamRepository {
	return &teamRepo{db: db}
}

func (r *teamRepo) Create(ctx context.Context, team *model.Team) (int64, error) {
	query := `INSERT INTO teams (name, description, created_by) VALUES (?, ?, ?)`
	result, err := r.db.ExecContext(ctx, query, team.Name, team.Description, team.CreatedBy)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (r *teamRepo) GetByID(ctx context.Context, id int64) (*model.Team, error) {
	query := `SELECT id, name, description, created_by, created_at, updated_at FROM teams WHERE id = ?`
	team := &model.Team{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&team.ID, &team.Name, &team.Description, &team.CreatedBy, &team.CreatedAt, &team.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return team, nil
}

func (r *teamRepo) ListByUser(ctx context.Context, userID int64) ([]model.Team, error) {
	query := `
		SELECT t.id, t.name, t.description, t.created_by, t.created_at, t.updated_at
		FROM teams t
		INNER JOIN team_members tm ON t.id = tm.team_id
		WHERE tm.user_id = ?
		ORDER BY t.created_at DESC`
	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var teams []model.Team
	for rows.Next() {
		var t model.Team
		if err := rows.Scan(&t.ID, &t.Name, &t.Description, &t.CreatedBy, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		teams = append(teams, t)
	}
	return teams, rows.Err()
}

func (r *teamRepo) AddMember(ctx context.Context, member *model.TeamMember) error {
	query := `INSERT INTO team_members (team_id, user_id, role) VALUES (?, ?, ?)`
	_, err := r.db.ExecContext(ctx, query, member.TeamID, member.UserID, member.Role)
	return err
}

func (r *teamRepo) GetMember(ctx context.Context, teamID, userID int64) (*model.TeamMember, error) {
	query := `SELECT id, team_id, user_id, role, joined_at FROM team_members WHERE team_id = ? AND user_id = ?`
	m := &model.TeamMember{}
	err := r.db.QueryRowContext(ctx, query, teamID, userID).Scan(&m.ID, &m.TeamID, &m.UserID, &m.Role, &m.JoinedAt)
	if err != nil {
		return nil, err
	}
	return m, nil
}

func (r *teamRepo) GetMemberRole(ctx context.Context, teamID, userID int64) (model.TeamRole, error) {
	query := `SELECT role FROM team_members WHERE team_id = ? AND user_id = ?`
	var role model.TeamRole
	err := r.db.QueryRowContext(ctx, query, teamID, userID).Scan(&role)
	return role, err
}

func (r *teamRepo) IsMember(ctx context.Context, teamID, userID int64) (bool, error) {
	query := `SELECT COUNT(*) FROM team_members WHERE team_id = ? AND user_id = ?`
	var count int
	err := r.db.QueryRowContext(ctx, query, teamID, userID).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
