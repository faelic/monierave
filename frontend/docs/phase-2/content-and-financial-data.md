# Content and Financial Data

## Voice

Monierave writes in plain, direct language:

- Use `Send money`, not `Execute transfer`.
- Use `Current posted balance`, not `Funds available`.
- Use `We could not confirm the result yet`, not `Transaction error`.
- Use `Close account`, not `Delete account`.
- Use `Recipient`, not `counterparty`, in customer actions.

Tone remains calm during failure. Avoid blame, jokes, urgency theater, and
unearned reassurance.

## Page headings

Headings describe the user's location or decision:

```text
Accounts
Send money
Review transfer
Transfer posted
Close USD account
Verify your email
```

Avoid greetings as the only page heading. `Good morning` may be supporting copy
but does not replace `Overview`.

## Action labels

Buttons state the effect:

```text
Continue to review
Send USD 50.00
Save beneficiary
Resend verification email
Close account
Sign out everywhere
```

Avoid `Submit`, `Proceed`, `Okay`, and ambiguous `Confirm` where a precise verb
fits.

## Amount formatting

- Backend values are integer minor units.
- Format with `Intl.NumberFormat` using explicit currency and user locale.
- Conversion from form major units to minor units uses decimal-safe parsing,
  never binary floating-point multiplication.
- Use the currency's actual fraction digits.
- Include the ISO currency when symbol ambiguity matters.
- Keep signs semantic: outgoing `-`, incoming `+` where useful.
- Screen-reader text includes direction, amount, and currency.

Examples:

```text
USD 1,250.00
-EUR 42.50
Incoming: USD 500.00
```

## Balance language

The backend models a current posted balance only. Approved labels:

- `Current posted balance`
- `Posted balance`, in compact supporting contexts

Prohibited labels:

- `Available balance`
- `Spendable balance`
- `Withdrawable balance`

Do not imply that pending card authorizations, holds, settlement, or external
rails are represented.

## Account identity

Owned account detail may show the full account number with an explicit copy
control. Lists and breadcrumbs default to:

```text
USD account
******1756
```

Recipient and beneficiary views always remain masked. Public UUIDs may appear
in URLs but are not customer-facing labels.

## Transaction status language

| Backend state            | Customer label  | Meaning                                        |
| ------------------------ | --------------- | ---------------------------------------------- |
| `pending`                | Pending         | Accepted but not confirmed as posted           |
| `posted`                 | Posted          | Confirmed in the ledger                        |
| `failed`                 | Failed          | Confirmed not completed                        |
| `reversed`               | Reversed        | Offset by a linked reversal                    |
| Unknown transport result | Checking result | Client does not yet know; not a backend status |

`Checking result` is an interface state and must not be written back as a
transaction status.

## Error content

Error messages follow:

```text
What happened.
What remains true.
What the user can do next.
```

Examples:

### Insufficient funds

```text
This account does not have enough posted funds for the transfer.
Choose a smaller amount or another account.
```

### Frozen sender

```text
This account is frozen and cannot send money.
Choose another active account.
```

### Recipient not found

```text
We could not find a recipient that can receive this transfer.
Check the account number and try again.
```

Do not explain whether the destination is closed or nonexistent.

### Uncertain transfer

```text
We could not confirm the transfer result yet.
Do not send it again. We will check the original request.
```

The retry mechanism reuses the same idempotency key.

## Dates and time

- Use the user's locale and timezone for display.
- Preserve the absolute timestamp in machine-readable markup or detail.
- Relative time may supplement, not replace, the exact time on financial
  records.
- Statements show explicit date ranges.
- Avoid ambiguous numeric-only dates.

## References and identifiers

- Transaction references are selectable and copyable.
- Do not truncate a reference where users may need support.
- UUIDs are not shown unless an operational support requirement is approved.
- Copy events never enter analytics with the copied value.

## Privacy display rules

- Passwords and verification tokens are never echoed.
- Recipient names remain masked exactly as returned.
- Beneficiary destination numbers remain masked.
- Email may be partially masked in verification guidance when full display is
  unnecessary.
- Session and future card details follow least-display principles.
- Sensitive values never appear in toast titles, URLs, page titles, or browser
  notifications.

## Content state checklist

Every page or component specification must include copy for:

- First load.
- Empty state.
- Validation error.
- Business-rule rejection.
- Dependency failure.
- Offline state.
- Success.
- Partial or uncertain result where applicable.
- Session expiration.
- Restricted capability.

Placeholder lorem ipsum is not accepted in security or financial flows.
