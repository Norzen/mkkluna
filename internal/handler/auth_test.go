package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"taskmanager/internal/model"
	"taskmanager/internal/service"
)

type mockAuthService struct {
	registerFn func(ctx context.Context, req model.RegisterRequest) (*model.AuthResponse, error)
	loginFn    func(ctx context.Context, req model.LoginRequest) (*model.AuthResponse, error)
}

func (m *mockAuthService) Register(ctx context.Context, req model.RegisterRequest) (*model.AuthResponse, error) {
	return m.registerFn(ctx, req)
}
func (m *mockAuthService) Login(ctx context.Context, req model.LoginRequest) (*model.AuthResponse, error) {
	return m.loginFn(ctx, req)
}
func (m *mockAuthService) ValidateToken(_ string) (int64, error) { return 0, nil }

func TestAuthHandler_Register_Success(t *testing.T) {
	svc := &mockAuthService{
		registerFn: func(_ context.Context, req model.RegisterRequest) (*model.AuthResponse, error) {
			return &model.AuthResponse{
				Token: "test-token",
				User: model.User{
					ID: 1, Email: req.Email, Name: req.Name,
					CreatedAt: time.Now(), UpdatedAt: time.Now(),
				},
			}, nil
		},
	}
	h := NewAuthHandler(svc)

	body, _ := json.Marshal(model.RegisterRequest{
		Email: "test@test.com", Password: "pass", Name: "Test",
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/register", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	h.Register(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rec.Code)
	}

	var resp model.AuthResponse
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp.Token != "test-token" {
		t.Fatalf("expected token 'test-token', got '%s'", resp.Token)
	}
}

func TestAuthHandler_Register_Conflict(t *testing.T) {
	svc := &mockAuthService{
		registerFn: func(_ context.Context, _ model.RegisterRequest) (*model.AuthResponse, error) {
			return nil, service.ErrEmailTaken
		},
	}
	h := NewAuthHandler(svc)

	body, _ := json.Marshal(model.RegisterRequest{Email: "dup@test.com", Password: "p", Name: "N"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/register", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	h.Register(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", rec.Code)
	}
}

func TestAuthHandler_Login_Success(t *testing.T) {
	svc := &mockAuthService{
		loginFn: func(_ context.Context, _ model.LoginRequest) (*model.AuthResponse, error) {
			return &model.AuthResponse{Token: "jwt-token", User: model.User{ID: 1}}, nil
		},
	}
	h := NewAuthHandler(svc)

	body, _ := json.Marshal(model.LoginRequest{Email: "test@test.com", Password: "pass"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/login", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	h.Login(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestAuthHandler_Login_Invalid(t *testing.T) {
	svc := &mockAuthService{
		loginFn: func(_ context.Context, _ model.LoginRequest) (*model.AuthResponse, error) {
			return nil, service.ErrInvalidCredentials
		},
	}
	h := NewAuthHandler(svc)

	body, _ := json.Marshal(model.LoginRequest{Email: "bad@test.com", Password: "wrong"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/login", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	h.Login(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestAuthHandler_Register_BadBody(t *testing.T) {
	svc := &mockAuthService{}
	h := NewAuthHandler(svc)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/register", bytes.NewReader([]byte("invalid")))
	rec := httptest.NewRecorder()

	h.Register(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestAuthHandler_Login_BadBody(t *testing.T) {
	svc := &mockAuthService{}
	h := NewAuthHandler(svc)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/login", bytes.NewReader([]byte("invalid")))
	rec := httptest.NewRecorder()

	h.Login(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestAuthHandler_Login_InternalError(t *testing.T) {
	svc := &mockAuthService{
		loginFn: func(_ context.Context, _ model.LoginRequest) (*model.AuthResponse, error) {
			return nil, errors.New("database error")
		},
	}
	h := NewAuthHandler(svc)

	body, _ := json.Marshal(model.LoginRequest{Email: "e@t.com", Password: "p"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/login", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	h.Login(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
}
