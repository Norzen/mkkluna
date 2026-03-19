package service

import (
	"errors"
	"testing"

	"taskmanager/internal/model"
)

func TestCreateTask_NoTeamID(t *testing.T) {
	svc := NewTaskService(newMockTaskRepo(), newMockTeamRepo(), &mockCache{})

	_, err := svc.Create(t.Context(), 1, model.CreateTaskRequest{Title: "T"})
	if err == nil {
		t.Fatal("expected error for missing team_id")
	}
}

func TestCreateTask_AssigneeNotMember(t *testing.T) {
	taskRepo := newMockTaskRepo()
	teamRepo := newMockTeamRepo()
	teamRepo.members = append(teamRepo.members, model.TeamMember{TeamID: 1, UserID: 1, Role: model.RoleMember})
	teamRepo.teams[1] = &model.Team{ID: 1, Name: "Team"}

	svc := NewTaskService(taskRepo, teamRepo, &mockCache{})

	assigneeID := int64(99)
	_, err := svc.Create(t.Context(), 1, model.CreateTaskRequest{
		Title: "T", TeamID: 1, AssigneeID: &assigneeID,
	})
	if err == nil {
		t.Fatal("expected error for assignee not a member")
	}
}

func TestCreateTask_WithAssignee(t *testing.T) {
	taskRepo := newMockTaskRepo()
	teamRepo := newMockTeamRepo()
	teamRepo.members = append(teamRepo.members,
		model.TeamMember{TeamID: 1, UserID: 1, Role: model.RoleMember},
		model.TeamMember{TeamID: 1, UserID: 2, Role: model.RoleMember},
	)
	teamRepo.teams[1] = &model.Team{ID: 1, Name: "Team"}

	svc := NewTaskService(taskRepo, teamRepo, &mockCache{})

	assigneeID := int64(2)
	task, err := svc.Create(t.Context(), 1, model.CreateTaskRequest{
		Title: "Assigned Task", TeamID: 1, AssigneeID: &assigneeID,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if task.AssigneeID == nil || *task.AssigneeID != 2 {
		t.Fatal("expected assignee_id to be 2")
	}
}

func TestCreateTask_WithCustomStatusAndPriority(t *testing.T) {
	taskRepo := newMockTaskRepo()
	teamRepo := newMockTeamRepo()
	teamRepo.members = append(teamRepo.members, model.TeamMember{TeamID: 1, UserID: 1, Role: model.RoleMember})
	teamRepo.teams[1] = &model.Team{ID: 1, Name: "Team"}

	svc := NewTaskService(taskRepo, teamRepo, &mockCache{})

	task, err := svc.Create(t.Context(), 1, model.CreateTaskRequest{
		Title:    "Custom",
		TeamID:   1,
		Status:   model.StatusInProgress,
		Priority: model.PriorityHigh,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if task.Status != model.StatusInProgress {
		t.Fatalf("expected in_progress, got %s", task.Status)
	}
	if task.Priority != model.PriorityHigh {
		t.Fatalf("expected high, got %s", task.Priority)
	}
}

func TestUpdateTask_ChangeAssignee(t *testing.T) {
	taskRepo := newMockTaskRepo()
	teamRepo := newMockTeamRepo()
	teamRepo.members = append(teamRepo.members,
		model.TeamMember{TeamID: 1, UserID: 1, Role: model.RoleMember},
		model.TeamMember{TeamID: 1, UserID: 2, Role: model.RoleMember},
	)
	teamRepo.teams[1] = &model.Team{ID: 1, Name: "Team"}
	svc := NewTaskService(taskRepo, teamRepo, &mockCache{})

	task, _ := svc.Create(t.Context(), 1, model.CreateTaskRequest{
		Title: "Task", TeamID: 1,
	})

	newAssignee := int64(2)
	updated, err := svc.Update(t.Context(), 1, task.ID, model.UpdateTaskRequest{
		AssigneeID: &newAssignee,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if updated.AssigneeID == nil || *updated.AssigneeID != 2 {
		t.Fatal("expected assignee to be updated to 2")
	}
}

func TestUpdateTask_ChangeAssigneeToNonMember(t *testing.T) {
	taskRepo := newMockTaskRepo()
	teamRepo := newMockTeamRepo()
	teamRepo.members = append(teamRepo.members, model.TeamMember{TeamID: 1, UserID: 1, Role: model.RoleMember})
	teamRepo.teams[1] = &model.Team{ID: 1, Name: "Team"}
	svc := NewTaskService(taskRepo, teamRepo, &mockCache{})

	task, _ := svc.Create(t.Context(), 1, model.CreateTaskRequest{
		Title: "Task", TeamID: 1,
	})

	badAssignee := int64(99)
	_, err := svc.Update(t.Context(), 1, task.ID, model.UpdateTaskRequest{
		AssigneeID: &badAssignee,
	})
	if err == nil {
		t.Fatal("expected error for non-member assignee")
	}
}

func TestUpdateTask_ChangeAllFields(t *testing.T) {
	taskRepo := newMockTaskRepo()
	teamRepo := newMockTeamRepo()
	teamRepo.members = append(teamRepo.members, model.TeamMember{TeamID: 1, UserID: 1, Role: model.RoleMember})
	teamRepo.teams[1] = &model.Team{ID: 1, Name: "Team"}
	svc := NewTaskService(taskRepo, teamRepo, &mockCache{})

	task, _ := svc.Create(t.Context(), 1, model.CreateTaskRequest{
		Title: "Original", Description: "Desc", TeamID: 1,
	})

	newTitle := "NewTitle"
	newDesc := "NewDesc"
	newStatus := model.StatusDone
	newPriority := model.PriorityLow
	updated, err := svc.Update(t.Context(), 1, task.ID, model.UpdateTaskRequest{
		Title:       &newTitle,
		Description: &newDesc,
		Status:      &newStatus,
		Priority:    &newPriority,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if updated.Title != "NewTitle" || updated.Description != "NewDesc" {
		t.Fatal("fields not updated correctly")
	}

	history, _ := svc.GetHistory(t.Context(), task.ID)
	if len(history) != 4 {
		t.Fatalf("expected 4 history entries, got %d", len(history))
	}
}

func TestListTasks_DefaultPagination(t *testing.T) {
	taskRepo := newMockTaskRepo()
	teamRepo := newMockTeamRepo()
	svc := NewTaskService(taskRepo, teamRepo, &mockCache{})

	resp, err := svc.List(t.Context(), model.TaskFilter{})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp.Page != 1 {
		t.Fatalf("expected page 1, got %d", resp.Page)
	}
	if resp.PageSize != 20 {
		t.Fatalf("expected page_size 20, got %d", resp.PageSize)
	}
}

func TestListTasks_WithFilter(t *testing.T) {
	taskRepo := newMockTaskRepo()
	teamRepo := newMockTeamRepo()
	teamRepo.members = append(teamRepo.members, model.TeamMember{TeamID: 1, UserID: 1, Role: model.RoleMember})
	teamRepo.teams[1] = &model.Team{ID: 1, Name: "Team"}
	svc := NewTaskService(taskRepo, teamRepo, &mockCache{})

	svc.Create(t.Context(), 1, model.CreateTaskRequest{Title: "T1", TeamID: 1, Status: model.StatusTodo})
	svc.Create(t.Context(), 1, model.CreateTaskRequest{Title: "T2", TeamID: 1, Status: model.StatusDone})

	resp, err := svc.List(t.Context(), model.TaskFilter{
		TeamID: 1, Status: model.StatusTodo, Page: 1, PageSize: 10,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp.TotalCount != 1 {
		t.Fatalf("expected 1 todo task, got %d", resp.TotalCount)
	}
}

func TestInviteUser_UserNotFound(t *testing.T) {
	teamRepo := newMockTeamRepo()
	userRepo := newMockUserRepo()
	emailSvc := &mockEmailSvc{}
	svc := NewTeamService(teamRepo, userRepo, emailSvc)

	team, _ := svc.Create(t.Context(), 1, model.CreateTeamRequest{Name: "Team"})

	err := svc.InviteUser(t.Context(), 1, team.ID, model.InviteRequest{
		UserID: 999, Role: model.RoleMember,
	})
	if !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("expected ErrUserNotFound, got %v", err)
	}
}

func TestInviteUser_NotMemberOfTeam(t *testing.T) {
	teamRepo := newMockTeamRepo()
	userRepo := newMockUserRepo()
	emailSvc := &mockEmailSvc{}
	svc := NewTeamService(teamRepo, userRepo, emailSvc)

	svc.Create(t.Context(), 1, model.CreateTeamRequest{Name: "Team"})

	err := svc.InviteUser(t.Context(), 99, 1, model.InviteRequest{
		UserID: 2, Role: model.RoleMember,
	})
	if !errors.Is(err, ErrNotMember) {
		t.Fatalf("expected ErrNotMember, got %v", err)
	}
}
