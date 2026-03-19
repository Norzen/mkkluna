package service

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"taskmanager/internal/model"
)

// --- Mock TeamRepository ---

type mockTeamRepo struct {
	teams   map[int64]*model.Team
	members []model.TeamMember
	nextID  int64
}

func newMockTeamRepo() *mockTeamRepo {
	return &mockTeamRepo{teams: make(map[int64]*model.Team), nextID: 1}
}

func (m *mockTeamRepo) Create(_ context.Context, team *model.Team) (int64, error) {
	team.ID = m.nextID
	m.nextID++
	stored := *team
	m.teams[team.ID] = &stored
	return team.ID, nil
}

func (m *mockTeamRepo) GetByID(_ context.Context, id int64) (*model.Team, error) {
	if t, ok := m.teams[id]; ok {
		return t, nil
	}
	return nil, sql.ErrNoRows
}

func (m *mockTeamRepo) ListByUser(_ context.Context, userID int64) ([]model.Team, error) {
	var result []model.Team
	for _, mem := range m.members {
		if mem.UserID == userID {
			if t, ok := m.teams[mem.TeamID]; ok {
				result = append(result, *t)
			}
		}
	}
	return result, nil
}

func (m *mockTeamRepo) AddMember(_ context.Context, member *model.TeamMember) error {
	for _, existing := range m.members {
		if existing.TeamID == member.TeamID && existing.UserID == member.UserID {
			return errors.New("duplicate")
		}
	}
	m.members = append(m.members, *member)
	return nil
}

func (m *mockTeamRepo) GetMember(_ context.Context, teamID, userID int64) (*model.TeamMember, error) {
	for _, mem := range m.members {
		if mem.TeamID == teamID && mem.UserID == userID {
			return &mem, nil
		}
	}
	return nil, sql.ErrNoRows
}

func (m *mockTeamRepo) GetMemberRole(_ context.Context, teamID, userID int64) (model.TeamRole, error) {
	for _, mem := range m.members {
		if mem.TeamID == teamID && mem.UserID == userID {
			return mem.Role, nil
		}
	}
	return "", sql.ErrNoRows
}

func (m *mockTeamRepo) IsMember(_ context.Context, teamID, userID int64) (bool, error) {
	for _, mem := range m.members {
		if mem.TeamID == teamID && mem.UserID == userID {
			return true, nil
		}
	}
	return false, nil
}

// --- Mock EmailService ---

type mockEmailSvc struct {
	sent []string
}

func (m *mockEmailSvc) SendInvitation(_ context.Context, email string, _ int64) error {
	m.sent = append(m.sent, email)
	return nil
}

// --- Tests ---

func TestCreateTeam_Success(t *testing.T) {
	teamRepo := newMockTeamRepo()
	userRepo := newMockUserRepo()
	emailSvc := &mockEmailSvc{}
	svc := NewTeamService(teamRepo, userRepo, emailSvc)

	team, err := svc.Create(t.Context(), 1, model.CreateTeamRequest{
		Name:        "Test Team",
		Description: "A test team",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if team.Name != "Test Team" {
		t.Fatalf("expected team name 'Test Team', got '%s'", team.Name)
	}
	if team.CreatedBy != 1 {
		t.Fatalf("expected created_by 1, got %d", team.CreatedBy)
	}

	isMember, _ := teamRepo.IsMember(t.Context(), team.ID, 1)
	if !isMember {
		t.Fatal("creator should be a member of the team")
	}

	role, _ := teamRepo.GetMemberRole(t.Context(), team.ID, 1)
	if role != model.RoleOwner {
		t.Fatalf("creator should have owner role, got %s", role)
	}
}

func TestCreateTeam_EmptyName(t *testing.T) {
	teamRepo := newMockTeamRepo()
	userRepo := newMockUserRepo()
	emailSvc := &mockEmailSvc{}
	svc := NewTeamService(teamRepo, userRepo, emailSvc)

	_, err := svc.Create(t.Context(), 1, model.CreateTeamRequest{Name: ""})
	if err == nil {
		t.Fatal("expected error for empty team name")
	}
}

func TestInviteUser_Success(t *testing.T) {
	teamRepo := newMockTeamRepo()
	userRepo := newMockUserRepo()
	emailSvc := &mockEmailSvc{}
	svc := NewTeamService(teamRepo, userRepo, emailSvc)

	userRepo.users["invitee@test.com"] = &model.User{ID: 2, Email: "invitee@test.com", Name: "Invitee"}

	team, _ := svc.Create(t.Context(), 1, model.CreateTeamRequest{Name: "Team"})

	err := svc.InviteUser(t.Context(), 1, team.ID, model.InviteRequest{
		UserID: 2, Role: model.RoleMember,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	isMember, _ := teamRepo.IsMember(t.Context(), team.ID, 2)
	if !isMember {
		t.Fatal("invited user should be a member")
	}

	if len(emailSvc.sent) != 1 || emailSvc.sent[0] != "invitee@test.com" {
		t.Fatal("expected email to be sent to invitee")
	}
}

func TestInviteUser_NotOwnerOrAdmin(t *testing.T) {
	teamRepo := newMockTeamRepo()
	userRepo := newMockUserRepo()
	emailSvc := &mockEmailSvc{}
	svc := NewTeamService(teamRepo, userRepo, emailSvc)

	team, _ := svc.Create(t.Context(), 1, model.CreateTeamRequest{Name: "Team"})

	userRepo.users["member@test.com"] = &model.User{ID: 3, Email: "member@test.com", Name: "Member"}
	teamRepo.AddMember(t.Context(), &model.TeamMember{TeamID: team.ID, UserID: 3, Role: model.RoleMember})

	userRepo.users["target@test.com"] = &model.User{ID: 4, Email: "target@test.com", Name: "Target"}

	err := svc.InviteUser(t.Context(), 3, team.ID, model.InviteRequest{
		UserID: 4, Role: model.RoleMember,
	})
	if !errors.Is(err, ErrNoPermission) {
		t.Fatalf("expected ErrNoPermission, got %v", err)
	}
}

func TestInviteUser_AlreadyMember(t *testing.T) {
	teamRepo := newMockTeamRepo()
	userRepo := newMockUserRepo()
	emailSvc := &mockEmailSvc{}
	svc := NewTeamService(teamRepo, userRepo, emailSvc)

	userRepo.users["existing@test.com"] = &model.User{ID: 2, Email: "existing@test.com", Name: "Existing"}

	team, _ := svc.Create(t.Context(), 1, model.CreateTeamRequest{Name: "Team"})
	_ = svc.InviteUser(t.Context(), 1, team.ID, model.InviteRequest{UserID: 2, Role: model.RoleMember})

	err := svc.InviteUser(t.Context(), 1, team.ID, model.InviteRequest{UserID: 2, Role: model.RoleMember})
	if !errors.Is(err, ErrAlreadyMember) {
		t.Fatalf("expected ErrAlreadyMember, got %v", err)
	}
}

func TestInviteUser_CannotInviteAsOwner(t *testing.T) {
	teamRepo := newMockTeamRepo()
	userRepo := newMockUserRepo()
	emailSvc := &mockEmailSvc{}
	svc := NewTeamService(teamRepo, userRepo, emailSvc)

	userRepo.users["target@test.com"] = &model.User{ID: 2, Email: "target@test.com", Name: "Target"}
	team, _ := svc.Create(t.Context(), 1, model.CreateTeamRequest{Name: "Team"})

	err := svc.InviteUser(t.Context(), 1, team.ID, model.InviteRequest{
		UserID: 2, Role: model.RoleOwner,
	})
	if err == nil {
		t.Fatal("expected error when inviting as owner")
	}
}

func TestListTeamsByUser(t *testing.T) {
	teamRepo := newMockTeamRepo()
	userRepo := newMockUserRepo()
	emailSvc := &mockEmailSvc{}
	svc := NewTeamService(teamRepo, userRepo, emailSvc)

	svc.Create(t.Context(), 1, model.CreateTeamRequest{Name: "Team1"})
	svc.Create(t.Context(), 1, model.CreateTeamRequest{Name: "Team2"})
	svc.Create(t.Context(), 2, model.CreateTeamRequest{Name: "Team3"})

	teams, err := svc.ListByUser(t.Context(), 1)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(teams) != 2 {
		t.Fatalf("expected 2 teams, got %d", len(teams))
	}
}
