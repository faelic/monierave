package api

import "errors"

var (
	ErrInvalidCredentials      = errors.New("invalid username or password")
	ErrUserAlreadyExists       = errors.New("user already exists")
	ErrUnauthorized            = errors.New("unauthorized")
	ErrForbidden               = errors.New("forbidden")
	ErrInternalServer          = errors.New("internal server error")
	ErrInvalidToken            = errors.New("invalid token")
	ErrExpiredToken            = errors.New("expired token")
	ErrBlockedSession          = errors.New("blocked session")
	ErrInvalidSession          = errors.New("invalid user session")
	ErrSessionMismatch         = errors.New("mismatched token session")
	ErrExpiredSession          = errors.New("expired session")
	ErrCurrentPasswordRequired = errors.New("current password is required to change password")

	ErrAccountNotFound     = errors.New("account not found")
	ErrCurrencyMismatch    = errors.New("account currency mismatch")
	ErrInsufficientBalance = errors.New("insufficient balance")
	ErrTransferFailed      = errors.New("transfer failed")
	ErrUserNotFound        = errors.New("user not found")

	ErrUsernameAlreadyExists     = errors.New("username already exists")
	ErrEmailAlreadyExists        = errors.New("email already exists")
	ErrVerficationEmailFailed    = errors.New("failed to send verification code")
	ErrEmailVerificationRequired = errors.New("email verification is required for financial features")
	ErrRegistrationExpired       = errors.New("email verification window expired; update your email or request another verification email")
	ErrEmailAlreadyVerified      = errors.New("email is already verified")
	ErrVerificationCooldown      = errors.New("wait before requesting another verification email")
)
