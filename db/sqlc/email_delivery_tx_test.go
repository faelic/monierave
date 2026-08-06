package db

import (
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
)

func TestCanApplyEmailDeliveryTransition(t *testing.T) {
	now := time.Now().UTC()
	tests := []struct {
		name    string
		current string
		next    string
		want    bool
	}{
		{name: "pending accepted", current: EmailDeliveryStatusPending, next: EmailDeliveryStatusAccepted, want: true},
		{name: "pending terminal", current: EmailDeliveryStatusPending, next: EmailDeliveryStatusBounced, want: true},
		{name: "accepted delayed", current: EmailDeliveryStatusAccepted, next: EmailDeliveryStatusDelayed, want: true},
		{name: "accepted delivered", current: EmailDeliveryStatusAccepted, next: EmailDeliveryStatusDelivered, want: true},
		{name: "delayed delivered", current: EmailDeliveryStatusDelayed, next: EmailDeliveryStatusDelivered, want: true},
		{name: "delivered complained", current: EmailDeliveryStatusDelivered, next: EmailDeliveryStatusComplained, want: true},
		{name: "delivered bounced rejected", current: EmailDeliveryStatusDelivered, next: EmailDeliveryStatusBounced},
		{name: "terminal delivered rejected", current: EmailDeliveryStatusBounced, next: EmailDeliveryStatusDelivered},
		{name: "same state rejected", current: EmailDeliveryStatusAccepted, next: EmailDeliveryStatusAccepted},
		{name: "unknown rejected", current: EmailDeliveryStatusPending, next: "unknown"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.want, canApplyEmailDeliveryTransition(
				test.current,
				test.next,
				pgtype.Timestamptz{},
				now,
			))
		})
	}
}

func TestEmailDeliveryTransitionRejectsStaleEvents(t *testing.T) {
	now := time.Now().UTC()
	currentAt := pgtype.Timestamptz{Time: now, Valid: true}

	require.False(t, canApplyEmailDeliveryTransition(
		EmailDeliveryStatusAccepted,
		EmailDeliveryStatusDelivered,
		currentAt,
		now,
	))
	require.False(t, canApplyEmailDeliveryTransition(
		EmailDeliveryStatusAccepted,
		EmailDeliveryStatusDelivered,
		currentAt,
		now.Add(-time.Second),
	))
	require.True(t, canApplyEmailDeliveryTransition(
		EmailDeliveryStatusAccepted,
		EmailDeliveryStatusDelivered,
		currentAt,
		now.Add(time.Second),
	))
}
