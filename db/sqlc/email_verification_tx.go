package db

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

const emailVerificationRequestCooldown = time.Minute

type VerifyUserEmailTxParams struct {
	Username string
	Email    string
	JobID    pgtype.UUID
}

func (store *SQLStore) VerifyUserEmailTx(
	ctx context.Context,
	arg VerifyUserEmailTxParams,
) (User, error) {
	var result User

	err := store.execTx(ctx, func(q *Queries) error {
		job, err := q.GetEmailJob(ctx, arg.JobID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrEmailVerificationJobMismatch
			}
			return err
		}
		if job.Username != arg.Username || !strings.EqualFold(job.Recipient, arg.Email) {
			return ErrEmailVerificationJobMismatch
		}

		user, err := q.GetUserForUpdate(ctx, arg.Username)
		if err != nil {
			return err
		}
		if !strings.EqualFold(user.Email, arg.Email) {
			return ErrEmailVerificationAddressStale
		}
		if user.AccountStatus == AccountStatusDisabled {
			return ErrRegistrationDisabled
		}
		if user.EmailDeliverabilityStatus == EmailDeliverabilityUndeliverable {
			return ErrEmailAddressUndeliverable
		}
		if user.AccountStatus == AccountStatusActive && user.EmailVerifiedAt.Valid {
			result = user
			return nil
		}

		result, err = q.MarkUserEmailVerified(ctx, MarkUserEmailVerifiedParams{
			Username: arg.Username,
			Email:    arg.Email,
		})
		if err != nil {
			return err
		}

		metadata, err := json.Marshal(map[string]string{})
		if err != nil {
			return err
		}
		_, err = q.CreateAuditLog(ctx, CreateAuditLogParams{
			EntityType:    "email_job",
			EntityID:      job.ID,
			CorrelationID: job.ID,
			EventType:     "email_verified",
			Actor:         "user",
			FromState:     nullableText(user.AccountStatus),
			ToState:       nullableText(AccountStatusActive),
			Metadata:      metadata,
		})
		return err
	})

	return result, err
}

type RequestEmailVerificationTxResult struct {
	User        User        `json:"user"`
	EmailJob    EmailJob    `json:"email_job"`
	OutboxEvent OutboxEvent `json:"outbox_event"`
}

func (store *SQLStore) RequestEmailVerificationTx(
	ctx context.Context,
	username string,
) (RequestEmailVerificationTxResult, error) {
	var result RequestEmailVerificationTxResult

	err := store.execTx(ctx, func(q *Queries) error {
		user, err := q.GetUserForUpdate(ctx, username)
		if err != nil {
			return err
		}
		if user.EmailVerifiedAt.Valid && user.AccountStatus == AccountStatusActive {
			return ErrEmailAlreadyVerified
		}
		if user.EmailDeliverabilityStatus == EmailDeliverabilityUndeliverable {
			return ErrEmailAddressUndeliverable
		}

		latest, latestErr := q.GetLatestEmailJobForCurrentAddress(ctx, username)
		if latestErr != nil && !errors.Is(latestErr, pgx.ErrNoRows) {
			return latestErr
		}
		if latestErr == nil &&
			latest.CreatedAt.Valid &&
			time.Since(latest.CreatedAt.Time) < emailVerificationRequestCooldown {
			return ErrEmailVerificationCooldown
		}

		result.User, err = q.RestartUserEmailVerification(ctx, username)
		if err != nil {
			return err
		}
		result.EmailJob, result.OutboxEvent, err = createVerificationEmailJob(
			ctx,
			q,
			result.User.Username,
			result.User.Email,
		)
		return err
	})

	return result, err
}
