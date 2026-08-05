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

func (store *SQLStore) UpdateUserTx(
	ctx context.Context,
	arg UpdateUserParams,
) (UpdateUserTxResult, error) {
	var result UpdateUserTxResult

	err := store.execTx(ctx, func(q *Queries) error {
		current, err := q.GetUser(ctx, arg.Username)
		if err != nil {
			return err
		}

		result.User, err = q.UpdateUser(ctx, arg)
		if err != nil {
			return err
		}

		result.EmailChanged = !strings.EqualFold(current.Email, result.User.Email)
		if !result.EmailChanged {
			return nil
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
