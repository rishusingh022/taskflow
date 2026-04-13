package service

import (
	"context"
	"testing"

	"taskflow/internal/model"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// --- mock user repo ---

type mockUserRepo struct {
	users map[string]*model.User
}

func newMockUserRepo() *mockUserRepo {
	return &mockUserRepo{users: make(map[string]*model.User)}
}

func (m *mockUserRepo) Create(ctx context.Context, user *model.User) error {
	user.ID = uuid.New()
	m.users[user.Email] = user
	return nil
}

func (m *mockUserRepo) FindByEmail(ctx context.Context, email string) (*model.User, error) {
	u, ok := m.users[email]
	if !ok {
		return nil, nil
	}
	return u, nil
}

func (m *mockUserRepo) FindByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	for _, u := range m.users {
		if u.ID == id {
			return u, nil
		}
	}
	return nil, nil
}

func TestAuthService_Register(t *testing.T) {
	repo := newMockUserRepo()
	svc := NewAuthService(repo, "test-secret")

	resp, err := svc.Register(context.Background(), model.RegisterRequest{
		Name:     "Test User",
		Email:    "test@example.com",
		Password: "password123",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Token == "" {
		t.Error("expected non-empty token")
	}
	if resp.User.Name != "Test User" {
		t.Errorf("expected name Test User, got %s", resp.User.Name)
	}
	if resp.User.Email != "test@example.com" {
		t.Errorf("expected email test@example.com, got %s", resp.User.Email)
	}
}

func TestAuthService_RegisterDuplicate(t *testing.T) {
	repo := newMockUserRepo()
	svc := NewAuthService(repo, "test-secret")

	req := model.RegisterRequest{Name: "User", Email: "dup@example.com", Password: "pass123"}
	_, err := svc.Register(context.Background(), req)
	if err != nil {
		t.Fatalf("first register failed: %v", err)
	}

	_, err = svc.Register(context.Background(), req)
	if err != model.ErrAlreadyExists {
		t.Errorf("expected ErrAlreadyExists, got %v", err)
	}
}

func TestAuthService_Login(t *testing.T) {
	repo := newMockUserRepo()
	svc := NewAuthService(repo, "test-secret")

	// register first
	_, err := svc.Register(context.Background(), model.RegisterRequest{
		Name:     "Login User",
		Email:    "login@example.com",
		Password: "mypassword",
	})
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}

	// login
	resp, err := svc.Login(context.Background(), model.LoginRequest{
		Email:    "login@example.com",
		Password: "mypassword",
	})
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}
	if resp.Token == "" {
		t.Error("expected token")
	}
}

func TestAuthService_LoginWrongPassword(t *testing.T) {
	repo := newMockUserRepo()
	svc := NewAuthService(repo, "test-secret")

	hashed, _ := bcrypt.GenerateFromPassword([]byte("correctpass"), 4) // low cost for speed
	repo.users["wrong@example.com"] = &model.User{
		ID:       uuid.New(),
		Name:     "User",
		Email:    "wrong@example.com",
		Password: string(hashed),
	}

	_, err := svc.Login(context.Background(), model.LoginRequest{
		Email:    "wrong@example.com",
		Password: "wrongpassword",
	})
	if err != model.ErrUnauthorized {
		t.Errorf("expected ErrUnauthorized, got %v", err)
	}
}

func TestAuthService_LoginUnknownEmail(t *testing.T) {
	repo := newMockUserRepo()
	svc := NewAuthService(repo, "test-secret")

	_, err := svc.Login(context.Background(), model.LoginRequest{
		Email:    "nobody@example.com",
		Password: "anything",
	})
	if err != model.ErrUnauthorized {
		t.Errorf("expected ErrUnauthorized, got %v", err)
	}
}
