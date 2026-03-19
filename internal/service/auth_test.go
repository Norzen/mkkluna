package service

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"taskmanager/internal/model"
)

// --- Mock UserRepository ---

type mockUserRepo struct {
	users  map[string]*model.User
	nextID int64
}

func newMockUserRepo() *mockUserRepo {
	return &mockUserRepo{users: make(map[string]*model.User), nextID: 1}
}

func (m *mockUserRepo) Create(_ context.Context, user *model.User) (int64, error) {
	if _, exists := m.users[user.Email]; exists {
		return 0, errors.New("duplicate email")
	}
	user.ID = m.nextID
	m.nextID++
	stored := *user
	m.users[user.Email] = &stored
	return user.ID, nil
}

func (m *mockUserRepo) GetByID(_ context.Context, id int64) (*model.User, error) {
	for _, u := range m.users {
		if u.ID == id {
			return u, nil
		}
	}
	return nil, sql.ErrNoRows
}

func (m *mockUserRepo) GetByEmail(_ context.Context, email string) (*model.User, error) {
	if u, ok := m.users[email]; ok {
		return u, nil
	}
	return nil, sql.ErrNoRows
}

// --- Tests ---

func TestRegister_Success(t *testing.T) {
	repo := newMockUserRepo()
	svc := NewAuthService(repo, "test-secret", 24*time.Hour)

	resp, err := svc.Register(t.Context(), model.RegisterRequest{
		Email:    "test@example.com",
		Password: "password123",
		Name:     "Test User",
	})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp.Token == "" {
		t.Fatal("expected non-empty token")
	}
	if resp.User.Email != "test@example.com" {
		t.Fatalf("expected email test@example.com, got %s", resp.User.Email)
	}
	if resp.User.Name != "Test User" {
		t.Fatalf("expected name Test User, got %s", resp.User.Name)
	}
}

func TestRegister_DuplicateEmail(t *testing.T) {
	repo := newMockUserRepo()
	svc := NewAuthService(repo, "test-secret", 24*time.Hour)

	_, err := svc.Register(t.Context(), model.RegisterRequest{
		Email: "dup@example.com", Password: "pass", Name: "User1",
	})
	if err != nil {
		t.Fatalf("first register failed: %v", err)
	}

	_, err = svc.Register(t.Context(), model.RegisterRequest{
		Email: "dup@example.com", Password: "pass2", Name: "User2",
	})
	if !errors.Is(err, ErrEmailTaken) {
		t.Fatalf("expected ErrEmailTaken, got %v", err)
	}
}

func TestRegister_EmptyFields(t *testing.T) {
	repo := newMockUserRepo()
	svc := NewAuthService(repo, "test-secret", 24*time.Hour)

	_, err := svc.Register(t.Context(), model.RegisterRequest{})
	if err == nil {
		t.Fatal("expected error for empty fields")
	}
}

func TestLogin_Success(t *testing.T) {
	repo := newMockUserRepo()
	svc := NewAuthService(repo, "test-secret", 24*time.Hour)

	_, err := svc.Register(t.Context(), model.RegisterRequest{
		Email: "login@example.com", Password: "password123", Name: "Login User",
	})
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}

	resp, err := svc.Login(t.Context(), model.LoginRequest{
		Email: "login@example.com", Password: "password123",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp.Token == "" {
		t.Fatal("expected non-empty token")
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	repo := newMockUserRepo()
	svc := NewAuthService(repo, "test-secret", 24*time.Hour)

	_, _ = svc.Register(t.Context(), model.RegisterRequest{
		Email: "wp@example.com", Password: "correct", Name: "User",
	})

	_, err := svc.Login(t.Context(), model.LoginRequest{
		Email: "wp@example.com", Password: "wrong",
	})
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestLogin_UserNotFound(t *testing.T) {
	repo := newMockUserRepo()
	svc := NewAuthService(repo, "test-secret", 24*time.Hour)

	_, err := svc.Login(t.Context(), model.LoginRequest{
		Email: "nonexistent@example.com", Password: "pass",
	})
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestValidateToken_Success(t *testing.T) {
	repo := newMockUserRepo()
	svc := NewAuthService(repo, "test-secret", 24*time.Hour)

	resp, _ := svc.Register(t.Context(), model.RegisterRequest{
		Email: "tok@example.com", Password: "pass", Name: "User",
	})

	userID, err := svc.ValidateToken(resp.Token)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if userID != resp.User.ID {
		t.Fatalf("expected user ID %d, got %d", resp.User.ID, userID)
	}
}

func TestValidateToken_Invalid(t *testing.T) {
	repo := newMockUserRepo()
	svc := NewAuthService(repo, "test-secret", 24*time.Hour)

	_, err := svc.ValidateToken("invalid-token")
	if err == nil {
		t.Fatal("expected error for invalid token")
	}
}

func TestValidateToken_WrongSecret(t *testing.T) {
	repo := newMockUserRepo()
	svc1 := NewAuthService(repo, "secret1", 24*time.Hour)
	svc2 := NewAuthService(repo, "secret2", 24*time.Hour)

	resp, _ := svc1.Register(t.Context(), model.RegisterRequest{
		Email: "ws@example.com", Password: "pass", Name: "User",
	})

	_, err := svc2.ValidateToken(resp.Token)
	if err == nil {
		t.Fatal("expected error for wrong secret")
	}
}
