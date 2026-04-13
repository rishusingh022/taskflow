package model

import "errors"

var (
	ErrNotFound      = errors.New("not found")
	ErrForbidden     = errors.New("forbidden")
	ErrAlreadyExists = errors.New("already exists")
	ErrInvalidInput  = errors.New("validation failed")
	ErrUnauthorized  = errors.New("unauthorized")
)
