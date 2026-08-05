package db

import (
	"context"
	"strings"
)

type UpdateUserTxResult struct {
	User         User        `json:"user"`
	EmailChanged bool        `json:"email_changed"`
	EmailJob     EmailJob    `json:"email_job"`
	OutboxEvent  OutboxEvent `json:"outbox_event"`
}

type UpdateUserTxParams struct {
	UpdateUserParams
	RevokeSessions bool
}

func (store *SQLStore) UpdateUserTx(
	ctx context.Context,
	arg UpdateUserTxParams,
) (UpdateUserTxResult, error) {
	var result UpdateUserTxResult

	err := store.execTx(ctx, func(q *Queries) error {
		current, err := q.GetUser(ctx, arg.Username)
		if err != nil {
			return err
		}

		result.User, err = q.UpdateUser(ctx, arg.UpdateUserParams)
		if err != nil {
			return err
		}

		result.EmailChanged = !strings.EqualFold(current.Email, result.User.Email)
		if result.EmailChanged {
			result.EmailJob, result.OutboxEvent, err = createVerificationEmailJob(
				ctx,
				q,
				result.User.Username,
				result.User.Email,
			)
			if err != nil {
				return err
			}
		}

		if !arg.RevokeSessions {
			return nil
		}
		sessions, err := q.RevokeAllUserSessions(ctx, RevokeAllUserSessionsParams{
			Username:      result.User.Username,
			RevokedReason: textValue("password_changed"),
		})
		if err != nil {
			return err
		}
		for _, session := range sessions {
			if err := createSessionAudit(
				ctx,
				q,
				session,
				"sessions_revoked_password_change",
				"user",
				"active",
				"revoked",
			); err != nil {
				return err
			}
		}
		return nil
	})

	return result, err
}
