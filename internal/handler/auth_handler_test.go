package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"taskflow/internal/handler"
	"taskflow/internal/model"
	"taskflow/internal/service"
	"taskflow/internal/repository"

	"github.com/jmoiron/sqlx"
)

// stubDB returns nil — we'll use mocks at the service level for unit tests.
// Integration tests use a real DB.

type mockUserRepo struct {
	users map[string]*model.User
}

func TestAuthHandler_Register_Success(t *testing.T) {
	// For handler-level tests we test the HTTP contract directly.
	// Service layer is tested separately.
	t.Skip("requires wired service — see integration tests")
}

func TestAuthHandler_Register_ValidationError(t *testing.T) {
	// Build a real handler with nil deps — validation runs before service call
	authSvc := service.NewAuthService(repository.NewUserRepo(&sqlx.DB{}), "test-secret")
	h := handler.NewAuthHandler(authSvc)

	body := map[string]string{
		"name":  "",
		"email": "",
	}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.Register(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &resp)

	if resp["error"] != "validation failed" {
		t.Errorf("expected validation failed error, got %v", resp["error"])
	}

	fields, ok := resp["fields"].(map[string]interface{})
	if !ok {
		t.Fatal("expected fields in response")
	}
	if _, exists := fields["name"]; !exists {
		t.Error("expected name field error")
	}
	if _, exists := fields["email"]; !exists {
		t.Error("expected email field error")
	}
}

func TestAuthHandler_Login_ValidationError(t *testing.T) {
	authSvc := service.NewAuthService(repository.NewUserRepo(&sqlx.DB{}), "test-secret")
	h := handler.NewAuthHandler(authSvc)

	body := map[string]string{}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.Login(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestAuthHandler_Register_InvalidJSON(t *testing.T) {
	authSvc := service.NewAuthService(repository.NewUserRepo(&sqlx.DB{}), "test-secret")
	h := handler.NewAuthHandler(authSvc)

	req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader([]byte("not json")))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.Register(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestAuthHandler_Login_InvalidJSON(t *testing.T) {
	authSvc := service.NewAuthService(repository.NewUserRepo(&sqlx.DB{}), "test-secret")
	h := handler.NewAuthHandler(authSvc)

	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader([]byte("{bad")))
	rec := httptest.NewRecorder()

	h.Login(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}
