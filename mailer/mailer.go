package mailer

import (
	"context"
	"encoding/json"
	"errors"
)

type VerificationEmail struct {
	JobID     string
	Username  string
	Recipient string
	Payload   json.RawMessage
}

type FinancialNotificationEmail struct {
	JobID     string
	Username  string
	Recipient string
	Payload   json.RawMessage
}

type Mailer interface {
	SendVerificationEmail(ctx context.Context, message VerificationEmail) (string, error)
	SendFinancialNotification(
		ctx context.Context,
		message FinancialNotificationEmail,
	) (string, error)
}

type PermanentError struct {
	Err error
}

func (e *PermanentError) Error() string {
	return e.Err.Error()
}

func (e *PermanentError) Unwrap() error {
	return e.Err
}

func NewPermanentError(err error) error {
	return &PermanentError{Err: err}
}

func IsPermanent(err error) bool {
	var permanent *PermanentError
	return errors.As(err, &permanent)
}
