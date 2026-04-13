package config_test

import (
	"os"
	"testing"

	"taskflow/internal/config"
)

func TestLoad_MissingJWTSecret(t *testing.T) {
	os.Unsetenv("JWT_SECRET")

	_, err := config.Load()
	if err == nil {
		t.Error("expected error when JWT_SECRET is missing")
	}
}

func TestLoad_WithJWTSecret(t *testing.T) {
	os.Setenv("JWT_SECRET", "my-test-secret")
	defer os.Unsetenv("JWT_SECRET")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.JWTSecret != "my-test-secret" {
		t.Errorf("expected 'my-test-secret', got '%s'", cfg.JWTSecret)
	}
	if cfg.Port != "8080" {
		t.Errorf("expected default port '8080', got '%s'", cfg.Port)
	}
}

func TestLoad_CustomPort(t *testing.T) {
	os.Setenv("JWT_SECRET", "secret")
	os.Setenv("PORT", "9090")
	defer os.Unsetenv("JWT_SECRET")
	defer os.Unsetenv("PORT")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Port != "9090" {
		t.Errorf("expected port '9090', got '%s'", cfg.Port)
	}
}

func TestLoad_CustomDatabaseURL(t *testing.T) {
	os.Setenv("JWT_SECRET", "secret")
	os.Setenv("DATABASE_URL", "postgres://custom@localhost/mydb")
	defer os.Unsetenv("JWT_SECRET")
	defer os.Unsetenv("DATABASE_URL")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.DatabaseURL != "postgres://custom@localhost/mydb" {
		t.Errorf("unexpected DATABASE_URL: %s", cfg.DatabaseURL)
	}
}
