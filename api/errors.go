package api

import "errors"

var (
	ErrInvalidCredentials = errors.New("invalid username or password")
	ErrUserAlreadyExists  = errors.New("user already exists")
	ErrUnauthorized       = errors.New("unauthorized")
	ErrForbidden          = errors.New("forbidden")
	ErrInternalServer     = errors.New("internal server error")
	ErrInvalidToken       = errors.New("invalid token")
	ErrExpiredToken       = errors.New("expired token")
	ErrBlockedSession     = errors.New("blocked session")
	ErrInvalidSession     = errors.New("invalid user session")
	ErrSessionMismatch    = errors.New("mismatched token session")
	ErrExpiredSession     = errors.New("expired session")

	ErrAccountNotFound     = errors.New("account not found")
	ErrCurrencyMismatch    = errors.New("account currency mismatch")
	ErrInsufficientBalance = errors.New("insufficient balance")
	ErrTransferFailed      = errors.New("transfer failed")
)
