package model

import (
	"regexp"
	"time"

	"github.com/google/uuid"
)

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

type User struct {
	ID        uuid.UUID `json:"id" db:"id"`
	Name      string    `json:"name" db:"name"`
	Email     string    `json:"email" db:"email"`
	Password  string    `json:"-" db:"password"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

type RegisterRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (r *RegisterRequest) Validate() map[string]string {
	errs := make(map[string]string)
	if r.Name == "" {
		errs["name"] = "is required"
	}
	if r.Email == "" {
		errs["email"] = "is required"
	} else if !emailRegex.MatchString(r.Email) {
		errs["email"] = "must be a valid email address"
	}
	if r.Password == "" {
		errs["password"] = "is required"
	} else if len(r.Password) < 6 {
		errs["password"] = "must be at least 6 characters"
	}
	return errs
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (r *LoginRequest) Validate() map[string]string {
	errs := make(map[string]string)
	if r.Email == "" {
		errs["email"] = "is required"
	}
	if r.Password == "" {
		errs["password"] = "is required"
	}
	return errs
}

type AuthResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}
