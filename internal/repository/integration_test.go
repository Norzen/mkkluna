package repository

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"taskmanager/internal/model"
)

var testDB *sql.DB

func TestMain(m *testing.M) {
	ctx := context.Background()

	mysqlContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "mysql:8.0",
			ExposedPorts: []string{"3306/tcp"},
			Env: map[string]string{
				"MYSQL_ROOT_PASSWORD": "testroot",
				"MYSQL_DATABASE":      "testdb",
				"MYSQL_USER":          "testuser",
				"MYSQL_PASSWORD":      "testpass",
			},
			WaitingFor: wait.ForLog("ready for connections").
				WithOccurrence(2).
				WithStartupTimeout(120 * time.Second),
		},
		Started: true,
	})
	if err != nil {
		log.Fatalf("failed to start mysql container: %v", err)
	}

	defer mysqlContainer.Terminate(ctx)

	host, _ := mysqlContainer.Host(ctx)
	port, _ := mysqlContainer.MappedPort(ctx, "3306")

	dsn := fmt.Sprintf("testuser:testpass@tcp(%s:%s)/testdb?parseTime=true&loc=UTC&multiStatements=true", host, port.Port())

	testDB, err = sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("failed to connect to test db: %v", err)
	}

	for range 30 {
		if err := testDB.Ping(); err == nil {
			break
		}
		time.Sleep(time.Second)
	}

	migration, _ := os.ReadFile("../../migrations/001_init.sql")
	if _, err := testDB.Exec(string(migration)); err != nil {
		log.Fatalf("failed to run migration: %v", err)
	}

	code := m.Run()
	testDB.Close()
	os.Exit(code)
}

func cleanTables(t *testing.T) {
	t.Helper()
	tables := []string{"task_comments", "task_history", "tasks", "team_members", "teams", "users"}
	for _, table := range tables {
		testDB.Exec("DELETE FROM " + table)
	}
}

func TestUserRepository_Integration(t *testing.T) {
	cleanTables(t)
	repo := NewUserRepository(testDB)
	ctx := t.Context()

	id, err := repo.Create(ctx, &model.User{
		Email: "int@test.com", PasswordHash: "hash", Name: "Integration User",
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	if id == 0 {
		t.Fatal("expected non-zero id")
	}

	user, err := repo.GetByID(ctx, id)
	if err != nil {
		t.Fatalf("get by id: %v", err)
	}
	if user.Email != "int@test.com" {
		t.Fatalf("expected email int@test.com, got %s", user.Email)
	}

	user2, err := repo.GetByEmail(ctx, "int@test.com")
	if err != nil {
		t.Fatalf("get by email: %v", err)
	}
	if user2.ID != id {
		t.Fatalf("expected id %d, got %d", id, user2.ID)
	}

	_, err = repo.GetByEmail(ctx, "nonexistent@test.com")
	if err != sql.ErrNoRows {
		t.Fatalf("expected ErrNoRows, got %v", err)
	}
}

func TestTeamRepository_Integration(t *testing.T) {
	cleanTables(t)
	userRepo := NewUserRepository(testDB)
	teamRepo := NewTeamRepository(testDB)
	ctx := t.Context()

	userID, _ := userRepo.Create(ctx, &model.User{
		Email: "team@test.com", PasswordHash: "hash", Name: "Team User",
	})

	teamID, err := teamRepo.Create(ctx, &model.Team{
		Name: "Test Team", Description: "Integration test", CreatedBy: userID,
	})
	if err != nil {
		t.Fatalf("create team: %v", err)
	}

	err = teamRepo.AddMember(ctx, &model.TeamMember{
		TeamID: teamID, UserID: userID, Role: model.RoleOwner,
	})
	if err != nil {
		t.Fatalf("add member: %v", err)
	}

	teams, err := teamRepo.ListByUser(ctx, userID)
	if err != nil {
		t.Fatalf("list teams: %v", err)
	}
	if len(teams) != 1 {
		t.Fatalf("expected 1 team, got %d", len(teams))
	}

	isMember, err := teamRepo.IsMember(ctx, teamID, userID)
	if err != nil {
		t.Fatalf("is member: %v", err)
	}
	if !isMember {
		t.Fatal("expected user to be a member")
	}

	role, err := teamRepo.GetMemberRole(ctx, teamID, userID)
	if err != nil {
		t.Fatalf("get role: %v", err)
	}
	if role != model.RoleOwner {
		t.Fatalf("expected owner role, got %s", role)
	}
}

func TestTaskRepository_Integration(t *testing.T) {
	cleanTables(t)
	userRepo := NewUserRepository(testDB)
	teamRepo := NewTeamRepository(testDB)
	taskRepo := NewTaskRepository(testDB)
	ctx := t.Context()

	userID, _ := userRepo.Create(ctx, &model.User{
		Email: "task@test.com", PasswordHash: "hash", Name: "Task User",
	})
	teamID, _ := teamRepo.Create(ctx, &model.Team{
		Name: "Task Team", CreatedBy: userID,
	})
	teamRepo.AddMember(ctx, &model.TeamMember{
		TeamID: teamID, UserID: userID, Role: model.RoleOwner,
	})

	taskID, err := taskRepo.Create(ctx, &model.Task{
		Title: "Integration Task", Description: "Test desc",
		Status: model.StatusTodo, Priority: model.PriorityHigh,
		TeamID: teamID, CreatedBy: userID,
	})
	if err != nil {
		t.Fatalf("create task: %v", err)
	}

	task, err := taskRepo.GetByID(ctx, taskID)
	if err != nil {
		t.Fatalf("get task: %v", err)
	}
	if task.Title != "Integration Task" {
		t.Fatalf("expected 'Integration Task', got '%s'", task.Title)
	}

	task.Status = model.StatusDone
	err = taskRepo.Update(ctx, task)
	if err != nil {
		t.Fatalf("update task: %v", err)
	}

	updated, _ := taskRepo.GetByID(ctx, taskID)
	if updated.Status != model.StatusDone {
		t.Fatalf("expected status done, got %s", updated.Status)
	}

	err = taskRepo.AddHistory(ctx, &model.TaskHistory{
		TaskID: taskID, ChangedBy: userID,
		FieldName: "status", OldValue: "todo", NewValue: "done",
	})
	if err != nil {
		t.Fatalf("add history: %v", err)
	}

	history, err := taskRepo.GetHistory(ctx, taskID)
	if err != nil {
		t.Fatalf("get history: %v", err)
	}
	if len(history) != 1 {
		t.Fatalf("expected 1 history entry, got %d", len(history))
	}

	tasks, total, err := taskRepo.List(ctx, model.TaskFilter{TeamID: teamID, Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("list tasks: %v", err)
	}
	if total != 1 {
		t.Fatalf("expected total 1, got %d", total)
	}
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(tasks))
	}
}

func TestAnalyticsRepository_Integration(t *testing.T) {
	cleanTables(t)
	userRepo := NewUserRepository(testDB)
	teamRepo := NewTeamRepository(testDB)
	taskRepo := NewTaskRepository(testDB)
	analyticsRepo := NewAnalyticsRepository(testDB)
	ctx := t.Context()

	user1ID, _ := userRepo.Create(ctx, &model.User{Email: "a1@test.com", PasswordHash: "h", Name: "User1"})
	user2ID, _ := userRepo.Create(ctx, &model.User{Email: "a2@test.com", PasswordHash: "h", Name: "User2"})

	teamID, _ := teamRepo.Create(ctx, &model.Team{Name: "Analytics Team", CreatedBy: user1ID})
	teamRepo.AddMember(ctx, &model.TeamMember{TeamID: teamID, UserID: user1ID, Role: model.RoleOwner})
	teamRepo.AddMember(ctx, &model.TeamMember{TeamID: teamID, UserID: user2ID, Role: model.RoleMember})

	for i := range 3 {
		taskRepo.Create(ctx, &model.Task{
			Title: fmt.Sprintf("Task %d", i), Status: model.StatusDone,
			Priority: model.PriorityMedium, TeamID: teamID, CreatedBy: user1ID,
		})
	}
	taskRepo.Create(ctx, &model.Task{
		Title: "Task by user2", Status: model.StatusTodo,
		Priority: model.PriorityLow, TeamID: teamID, CreatedBy: user2ID,
	})

	stats, err := analyticsRepo.GetTeamStats(ctx)
	if err != nil {
		t.Fatalf("get team stats: %v", err)
	}
	if len(stats) == 0 {
		t.Fatal("expected at least 1 team stat")
	}
	found := false
	for _, s := range stats {
		if s.TeamID == teamID {
			found = true
			if s.MemberCount != 2 {
				t.Fatalf("expected 2 members, got %d", s.MemberCount)
			}
		}
	}
	if !found {
		t.Fatal("team not found in stats")
	}

	creators, err := analyticsRepo.GetTopCreators(ctx)
	if err != nil {
		t.Fatalf("get top creators: %v", err)
	}
	if len(creators) == 0 {
		t.Fatal("expected at least 1 top creator")
	}

	orphaned, err := analyticsRepo.GetOrphanedTasks(ctx)
	if err != nil {
		t.Fatalf("get orphaned tasks: %v", err)
	}
	if len(orphaned) != 0 {
		t.Fatalf("expected 0 orphaned tasks, got %d", len(orphaned))
	}

	user3ID, _ := userRepo.Create(ctx, &model.User{Email: "a3@test.com", PasswordHash: "h", Name: "Outsider"})
	taskRepo.Create(ctx, &model.Task{
		Title: "Orphan Task", Status: model.StatusTodo,
		Priority: model.PriorityHigh, TeamID: teamID, CreatedBy: user1ID,
		AssigneeID: &user3ID,
	})

	orphaned, err = analyticsRepo.GetOrphanedTasks(ctx)
	if err != nil {
		t.Fatalf("get orphaned tasks: %v", err)
	}
	if len(orphaned) != 1 {
		t.Fatalf("expected 1 orphaned task, got %d", len(orphaned))
	}
}
