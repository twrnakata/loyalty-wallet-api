package domain

import "errors"

var (
	ErrInvalidInput         = errors.New("invalid input")
	ErrUnauthorized         = errors.New("unauthorized")
	ErrNotFound             = errors.New("not found")
	ErrConflict             = errors.New("conflict")
	ErrInsufficientBalance  = errors.New("insufficient balance")
	ErrInvalidAmount        = errors.New("amount must be greater than zero")
	ErrMissingIdempotencyKey = errors.New("idempotency key is required")
	ErrEmailTaken           = errors.New("email already registered")
	ErrInvalidCredentials   = errors.New("invalid credentials")
	ErrWalletExists         = errors.New("wallet already exists")
)
