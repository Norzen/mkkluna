package service

import (
	"context"
	"database/sql"
	"errors"

	"taskmanager/internal/model"
	"taskmanager/internal/repository"
)

var (
	ErrTeamNotFound = errors.New("team not found")
	ErrNotMember    = errors.New("user is not a member of this team")
	ErrNoPermission = errors.New("insufficient permissions")
	ErrAlreadyMember = errors.New("user is already a member of this team")
)

type TeamService interface {
	Create(ctx context.Context, userID int64, req model.CreateTeamRequest) (*model.Team, error)
	ListByUser(ctx context.Context, userID int64) ([]model.Team, error)
	InviteUser(ctx context.Context, inviterID, teamID int64, req model.InviteRequest) error
}

type teamService struct {
	teamRepo repository.TeamRepository
	userRepo repository.UserRepository
	emailSvc EmailService
}

func NewTeamService(teamRepo repository.TeamRepository, userRepo repository.UserRepository, emailSvc EmailService) TeamService {
	return &teamService{
		teamRepo: teamRepo,
		userRepo: userRepo,
		emailSvc: emailSvc,
	}
}

func (s *teamService) Create(ctx context.Context, userID int64, req model.CreateTeamRequest) (*model.Team, error) {
	if req.Name == "" {
		return nil, errors.New("team name is required")
	}

	team := &model.Team{
		Name:        req.Name,
		Description: req.Description,
		CreatedBy:   userID,
	}

	id, err := s.teamRepo.Create(ctx, team)
	if err != nil {
		return nil, err
	}
	team.ID = id

	err = s.teamRepo.AddMember(ctx, &model.TeamMember{
		TeamID: id,
		UserID: userID,
		Role:   model.RoleOwner,
	})
	if err != nil {
		return nil, err
	}

	return team, nil
}

func (s *teamService) ListByUser(ctx context.Context, userID int64) ([]model.Team, error) {
	return s.teamRepo.ListByUser(ctx, userID)
}

func (s *teamService) InviteUser(ctx context.Context, inviterID, teamID int64, req model.InviteRequest) error {
	role, err := s.teamRepo.GetMemberRole(ctx, teamID, inviterID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotMember
		}
		return err
	}

	if role != model.RoleOwner && role != model.RoleAdmin {
		return ErrNoPermission
	}

	_, err = s.userRepo.GetByID(ctx, req.UserID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrUserNotFound
		}
		return err
	}

	isMember, err := s.teamRepo.IsMember(ctx, teamID, req.UserID)
	if err != nil {
		return err
	}
	if isMember {
		return ErrAlreadyMember
	}

	inviteRole := req.Role
	if inviteRole == "" {
		inviteRole = model.RoleMember
	}
	if inviteRole == model.RoleOwner {
		return errors.New("cannot invite as owner")
	}

	err = s.teamRepo.AddMember(ctx, &model.TeamMember{
		TeamID: teamID,
		UserID: req.UserID,
		Role:   inviteRole,
	})
	if err != nil {
		return err
	}

	invitee, _ := s.userRepo.GetByID(ctx, req.UserID)
	if invitee != nil {
		_ = s.emailSvc.SendInvitation(ctx, invitee.Email, teamID)
	}

	return nil
}
