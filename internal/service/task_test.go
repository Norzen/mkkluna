package service

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"taskmanager/internal/model"
)

// --- Mock TaskRepository ---

type mockTaskRepo struct {
	tasks   map[int64]*model.Task
	history []model.TaskHistory
	nextID  int64
}

func newMockTaskRepo() *mockTaskRepo {
	return &mockTaskRepo{tasks: make(map[int64]*model.Task), nextID: 1}
}

func (m *mockTaskRepo) Create(_ context.Context, task *model.Task) (int64, error) {
	task.ID = m.nextID
	m.nextID++
	stored := *task
	stored.CreatedAt = time.Now()
	stored.UpdatedAt = time.Now()
	m.tasks[task.ID] = &stored
	return task.ID, nil
}

func (m *mockTaskRepo) GetByID(_ context.Context, id int64) (*model.Task, error) {
	if t, ok := m.tasks[id]; ok {
		return t, nil
	}
	return nil, sql.ErrNoRows
}

func (m *mockTaskRepo) Update(_ context.Context, task *model.Task) error {
	if _, ok := m.tasks[task.ID]; !ok {
		return sql.ErrNoRows
	}
	stored := *task
	stored.UpdatedAt = time.Now()
	m.tasks[task.ID] = &stored
	return nil
}

func (m *mockTaskRepo) List(_ context.Context, filter model.TaskFilter) ([]model.Task, int64, error) {
	var result []model.Task
	for _, t := range m.tasks {
		match := true
		if filter.TeamID > 0 && t.TeamID != filter.TeamID {
			match = false
		}
		if filter.Status != "" && t.Status != filter.Status {
			match = false
		}
		if filter.AssigneeID > 0 && (t.AssigneeID == nil || *t.AssigneeID != filter.AssigneeID) {
			match = false
		}
		if match {
			result = append(result, *t)
		}
	}
	return result, int64(len(result)), nil
}

func (m *mockTaskRepo) AddHistory(_ context.Context, h *model.TaskHistory) error {
	m.history = append(m.history, *h)
	return nil
}

func (m *mockTaskRepo) GetHistory(_ context.Context, taskID int64) ([]model.TaskHistory, error) {
	var result []model.TaskHistory
	for _, h := range m.history {
		if h.TaskID == taskID {
			result = append(result, h)
		}
	}
	return result, nil
}

func (m *mockTaskRepo) AddComment(_ context.Context, _ *model.TaskComment) (int64, error) {
	return 1, nil
}

func (m *mockTaskRepo) GetComments(_ context.Context, _ int64) ([]model.TaskComment, error) {
	return nil, nil
}

// --- Mock Cache ---

type mockCache struct{}

func (m *mockCache) Get(_ context.Context, _ string, _ any) error {
	return errors.New("cache miss")
}
func (m *mockCache) Set(_ context.Context, _ string, _ any, _ time.Duration) error {
	return nil
}
func (m *mockCache) Delete(_ context.Context, _ ...string) error   { return nil }
func (m *mockCache) DeleteByPattern(_ context.Context, _ string) error { return nil }

// --- Tests ---

func TestCreateTask_Success(t *testing.T) {
	taskRepo := newMockTaskRepo()
	teamRepo := newMockTeamRepo()
	teamRepo.members = append(teamRepo.members, model.TeamMember{TeamID: 1, UserID: 1, Role: model.RoleMember})
	teamRepo.teams[1] = &model.Team{ID: 1, Name: "Team"}

	svc := NewTaskService(taskRepo, teamRepo, &mockCache{})

	task, err := svc.Create(t.Context(), 1, model.CreateTaskRequest{
		Title:  "Test Task",
		TeamID: 1,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if task.Title != "Test Task" {
		t.Fatalf("expected title 'Test Task', got '%s'", task.Title)
	}
	if task.Status != model.StatusTodo {
		t.Fatalf("expected default status todo, got %s", task.Status)
	}
	if task.Priority != model.PriorityMedium {
		t.Fatalf("expected default priority medium, got %s", task.Priority)
	}
}

func TestCreateTask_NotMember(t *testing.T) {
	taskRepo := newMockTaskRepo()
	teamRepo := newMockTeamRepo()
	teamRepo.teams[1] = &model.Team{ID: 1, Name: "Team"}

	svc := NewTaskService(taskRepo, teamRepo, &mockCache{})

	_, err := svc.Create(t.Context(), 99, model.CreateTaskRequest{
		Title:  "Test",
		TeamID: 1,
	})
	if !errors.Is(err, ErrNotMember) {
		t.Fatalf("expected ErrNotMember, got %v", err)
	}
}

func TestCreateTask_EmptyTitle(t *testing.T) {
	taskRepo := newMockTaskRepo()
	teamRepo := newMockTeamRepo()
	svc := NewTaskService(taskRepo, teamRepo, &mockCache{})

	_, err := svc.Create(t.Context(), 1, model.CreateTaskRequest{TeamID: 1})
	if err == nil {
		t.Fatal("expected error for empty title")
	}
}

func TestUpdateTask_Success(t *testing.T) {
	taskRepo := newMockTaskRepo()
	teamRepo := newMockTeamRepo()
	teamRepo.members = append(teamRepo.members, model.TeamMember{TeamID: 1, UserID: 1, Role: model.RoleMember})
	teamRepo.teams[1] = &model.Team{ID: 1, Name: "Team"}
	svc := NewTaskService(taskRepo, teamRepo, &mockCache{})

	task, _ := svc.Create(t.Context(), 1, model.CreateTaskRequest{
		Title: "Original", TeamID: 1,
	})

	newTitle := "Updated"
	newStatus := model.StatusInProgress
	updated, err := svc.Update(t.Context(), 1, task.ID, model.UpdateTaskRequest{
		Title:  &newTitle,
		Status: &newStatus,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if updated.Title != "Updated" {
		t.Fatalf("expected title 'Updated', got '%s'", updated.Title)
	}
	if updated.Status != model.StatusInProgress {
		t.Fatalf("expected status in_progress, got %s", updated.Status)
	}

	history, _ := svc.GetHistory(t.Context(), task.ID)
	if len(history) != 2 {
		t.Fatalf("expected 2 history entries, got %d", len(history))
	}
}

func TestUpdateTask_NotFound(t *testing.T) {
	taskRepo := newMockTaskRepo()
	teamRepo := newMockTeamRepo()
	svc := NewTaskService(taskRepo, teamRepo, &mockCache{})

	title := "New"
	_, err := svc.Update(t.Context(), 1, 999, model.UpdateTaskRequest{Title: &title})
	if !errors.Is(err, ErrTaskNotFound) {
		t.Fatalf("expected ErrTaskNotFound, got %v", err)
	}
}

func TestUpdateTask_NotMember(t *testing.T) {
	taskRepo := newMockTaskRepo()
	teamRepo := newMockTeamRepo()
	teamRepo.members = append(teamRepo.members, model.TeamMember{TeamID: 1, UserID: 1, Role: model.RoleMember})
	teamRepo.teams[1] = &model.Team{ID: 1, Name: "Team"}
	svc := NewTaskService(taskRepo, teamRepo, &mockCache{})

	task, _ := svc.Create(t.Context(), 1, model.CreateTaskRequest{
		Title: "Task", TeamID: 1,
	})

	title := "Hack"
	_, err := svc.Update(t.Context(), 99, task.ID, model.UpdateTaskRequest{Title: &title})
	if !errors.Is(err, ErrNotMember) {
		t.Fatalf("expected ErrNotMember, got %v", err)
	}
}

func TestListTasks(t *testing.T) {
	taskRepo := newMockTaskRepo()
	teamRepo := newMockTeamRepo()
	teamRepo.members = append(teamRepo.members, model.TeamMember{TeamID: 1, UserID: 1, Role: model.RoleMember})
	teamRepo.teams[1] = &model.Team{ID: 1, Name: "Team"}
	svc := NewTaskService(taskRepo, teamRepo, &mockCache{})

	svc.Create(t.Context(), 1, model.CreateTaskRequest{Title: "T1", TeamID: 1})
	svc.Create(t.Context(), 1, model.CreateTaskRequest{Title: "T2", TeamID: 1})

	resp, err := svc.List(t.Context(), model.TaskFilter{TeamID: 1, Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp.TotalCount != 2 {
		t.Fatalf("expected 2 tasks, got %d", resp.TotalCount)
	}
}
