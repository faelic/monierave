# Critical User Flows

## Flow conventions

Every financial flow must distinguish:

- `Preparing`: editable client-side intent.
- `Submitting`: one in-flight request; duplicate controls disabled.
- `Uncertain`: the request may have reached the server but no result is known.
- `Posted`: confirmed by the backend.
- `Failed`: confirmed rejection with a safe recovery action.

Analytics names below are recommendations. Events must exclude passwords,
tokens, full account numbers, names, emails, balances, and transfer narration.

## 1. Registration and email verification

**Entry:** `/signup`.

1. Collect username, full name, email, and password with visible requirements.
2. Validate locally, then submit `POST /users`.
3. On success, explain that the registration is pending and send the user to
   public `/signup/check-email`. Registration does not create an authenticated
   session.
4. The email link opens `GET /users/verify-email?token=...`, which validates
   intent but does not activate the account.
5. The confirmation action submits `POST /users/verify-email`.
6. Repeated confirmation of an already-used successful token remains a success.
7. After confirmation, direct the user to login; do not imply that verification
   itself created a session.

**Failure and recovery:** Field errors remain attached to inputs. Compromised
password and breach-check outage errors remain distinct. Expired, stale,
invalid, or disabled-registration tokens provide a safe next action. Permanent
email deliverability failure directs the user to change the email rather than
repeatedly resend.

**Security checkpoint:** Verification changes state only through POST. Tokens
are never sent to analytics or logs.

**Accessibility:** Focus moves to the error summary on failed submit. Delivery
and confirmation statuses are announced. Instructions do not depend on color.

**Analytics:** `registration_started`, `registration_submitted`,
`registration_created`, `verification_confirmed`, `verification_failed`.

## 2. Sign in and session restoration

**Entry:** `/login` or a protected deep link.

1. Submit username and password to `POST /users/login`.
2. Keep the access token in memory; secure refresh and device credentials stay
   in cookies controlled by the backend.
3. Fetch `GET /users/me`.
4. Route verified users to the safe return path or `/app`.
5. Route pending users to `/verification-needed`.
6. Route disabled registrations to explicit recovery guidance.

**Failure and recovery:** Invalid credentials remain neutral. Rate limiting
shows a wait state without identifying which credential was valid. On refresh,
attempt access renewal once, then restore `/users/me`; concurrent failed
requests must not trigger multiple refresh attempts.

**Security checkpoint:** Safe same-origin redirects only. Never place tokens in
URLs or local storage.

**Accessibility:** Password reveal has an accessible name and preserves focus.
Authentication failures are announced without clearing the username.

**Analytics:** `login_submitted`, `login_succeeded`, `login_failed`,
`session_restored`.

## 3. Pending-registration management

**Entry:** Financial route guard or `/verification-needed`.

1. Load current-user and email-status information.
2. Explain the exact verification-token lifetime separately from the
   registration grace period.
3. Allow resend when eligible.
4. Allow email correction through profile update.
5. After verification, refetch current user and continue to the original safe
   route.

**Failure and recovery:** Cooldown shows when another request is appropriate.
An undeliverable unchanged address requires an edit. Disabled registration
must not be described as merely waiting.

**Analytics:** `verification_status_viewed`, `verification_resend_requested`,
`verification_email_change_started`.

## 4. Create a financial account

**Entry:** Account empty state or `/app/accounts/new`.

1. Select one supported currency.
2. Explain that the account begins active with a zero current posted balance.
3. Submit only the currency to `POST /accounts`.
4. Display the backend-generated 10-digit account number after success.

**Validation:** Do not accept balance, UUID, status, or account number fields.
If an account already exists for the currency, offer navigation to it.

**Security checkpoint:** Email verification is required.

**Accessibility:** The generated number is selectable and has an explicit copy
button. Copy confirmation is announced without exposing the value in analytics.

**Analytics:** `account_creation_started`, `account_created`,
`account_creation_failed`, with currency only.

## 5. View balances, history, and statements

**Entry:** `/app`, accounts list, or an account deep link.

1. Fetch owned account data and label balance as `Current posted balance`.
2. Show lifecycle status beside the account identity.
3. Load account-scoped history with cursor pagination.
4. Preserve supported date, type, status, and direction filters in the URL.
5. Open transaction detail by public reference.

**Loading and empty:** Never render a temporary zero balance. Skeletons preserve
layout. No-activity states remain distinct from failed loading.

**Failure and recovery:** Foreign and missing UUIDs share a not-found state.
Stale cached data displays its last refresh time during connection loss.

**Accessibility:** Signed amounts include readable debit/credit language.
Tables have headers; mobile uses an equivalent list without losing semantics.

**Analytics:** `account_viewed`, `transaction_filter_applied`,
`statement_viewed`, using public route category but no account identifier.

## 6. Resolve a transfer recipient

**Entry:** `/app/transfers/new` or add-beneficiary flow.

1. Collect a 10-digit account number.
2. Submit `POST /accounts/resolve` only when the format is valid.
3. Show masked account number, masked name, currency, and receiving eligibility.
4. Require an explicit user choice to continue with the resolved recipient.

**Failure and recovery:** Malformed, missing, and closed destinations share
`recipient_not_found`. A `429` response pauses repeated resolution and shows a
generic retry message. Resolution never guarantees that transfer submission
will succeed.

**Security checkpoint:** Verified session required. Do not cache lookup results
persistently or send searched account numbers to analytics.

**Accessibility:** The masked identity is announced as the resolved recipient,
and the continue action includes that context in its accessible description.

**Analytics:** `recipient_resolution_requested`,
`recipient_resolution_succeeded`, `recipient_resolution_failed`, without
recipient data.

## 7. Create and use a beneficiary

**Entry:** Beneficiaries list or successful recipient resolution.

1. Resolve the full account number.
2. Collect and trim a nickname.
3. Submit `destination_account_number` and nickname to `POST /beneficiaries`.
4. Store only the beneficiary UUID and masked response in client caches.
5. Allow rename and remove with ownership-scoped beneficiary endpoints.

**Important limitation:** The current transfer contract accepts a destination
account number, while beneficiary responses expose only a masked number. The
frontend cannot initiate a transfer from a saved beneficiary after reload
without asking for the full number again. A backend transfer-by-beneficiary
contract is required before presenting one-tap beneficiary selection.

**Failure and recovery:** Closed destinations cannot be added. A beneficiary
whose destination later becomes ineligible remains visible with
`can_receive: false` and cannot be selected.

**Analytics:** `beneficiary_created`, `beneficiary_renamed`,
`beneficiary_removed`, without nickname or destination.

## 8. Prepare, review, and submit an internal transfer

**Entry:** `/app/transfers/new`.

1. Select an active owned sender account.
2. Resolve the 10-digit destination account number.
3. Enter a positive minor-unit amount through a currency-aware major-unit UI.
4. Enter optional narration and validate its length.
5. Show review with sender, masked recipient, amount, currency, and fee `0`.
6. State that limits, status, currency, and balance are rechecked on submit.
7. Generate an idempotency key only when the reviewed intent is confirmed.
8. Submit `POST /transfers` once and disable duplicate controls.

**Security checkpoint:** The review screen is non-editable. Editing returns to
the form and requires a new key. MFA is not claimed because no step-up contract
exists.

**Failure and recovery:** Map stable business errors to actionable guidance.
Insufficient funds returns to amount/sender selection. Frozen source requires a
different sender. Currency mismatch requires a matching destination. Limits
explain the rule without exposing security internals.

**Accessibility:** Review is a semantic definition list. Focus moves to the
result heading after submission. Amounts include currency and direction.

**Analytics:** `transfer_started`, `transfer_reviewed`,
`transfer_submitted`, `transfer_posted`, `transfer_rejected`, with error code,
currency, and amount band only.

## 9. Handle uncertain, failed, and replayed transfers

**Uncertain transport outcome:**

1. Keep the original intent and idempotency key in memory.
2. Do not label the transfer failed or restore a speculative balance.
3. Retry the exact request with the same key after connectivity/session
   recovery.
4. Treat `Idempotency-Replayed: true` as the original committed result.
5. Refetch sender account, history, and transaction detail.

**Confirmed failure:** Show the stable error and one recovery action. Any change
to transfer intent creates a new idempotency key.

**Pending state:** The data model supports statuses, but the current internal
transfer path normally returns a posted transaction. The UI architecture may
render pending status but must not simulate one.

**Analytics:** `transfer_outcome_uncertain`, `transfer_replay_attempted`,
`transfer_replay_resolved`.

## 10. Close an account

**Entry:** Account detail settings.

1. Explain that closure is permanent, preserves history, and preserves the
   account number.
2. Require the user to review currency, masked account number, and current
   posted balance.
3. Submit `POST /accounts/:public_id/close`.
4. On success, invalidate account lists and detail queries.

**Validation:** A non-zero balance or already-closed account returns conflict.
The UI never offers hard deletion.

**Security checkpoint:** Use a focused confirmation screen or dialog. Do not
rely on a generic toast for the final warning.

**Accessibility:** Initial focus lands on the warning heading, not the
destructive button. Confirmation wording states the consequence.

**Analytics:** `account_closure_started`, `account_closed`,
`account_closure_rejected`, without account identifiers.

## 11. Update profile or password

**Entry:** `/app/profile/edit`, available to pending and verified users.

1. Load safe current-user data.
2. Submit only changed fields to `PATCH /users/me`.
3. Require current password when changing password.
4. If email changes, reflect the reset verification/deliverability lifecycle.
5. Refetch current user and route pending users to verification guidance.

**Failure and recovery:** Password compromise and breach-check outage remain
distinct. Unknown fields are never sent. Preserve safe fields after validation
errors but always clear password fields.

**Analytics:** `profile_updated`, `password_change_succeeded`,
`profile_update_failed`, without changed values.

## 12. Logout and session expiration

**Logout current:** Call `POST /sessions/logout`, clear in-memory credentials and
all private query caches, then navigate to login.

**Logout all:** Confirm that all sessions, including the current one, will end;
call `POST /sessions/logout-all`; clear state and navigate to login.

**Session expiry:** Stop protected mutations, preserve a safe return path, and
require login. If expiry follows an in-flight transfer, resolve the outcome with
the original idempotency key after authentication.

**Individual device management:** Blocked until session-list and
single-revocation HTTP contracts exist.

**Analytics:** `logout_completed`, `logout_all_completed`, `session_expired`.

## Provisional future flows

The following flows define product intent only. They are blocked from
implementation until their backend contracts and security controls are
approved.

## 13. Recover an account

**Provisional entry:** `/forgot-password`.

1. Collect a normalized account identifier without revealing whether it exists.
2. Show the same accepted state for known and unknown users.
3. Deliver a single-use, short-lived recovery link through a verified channel.
4. Validate the token before showing the password form.
5. Apply the same password policy and breach check used by registration.
6. Revoke existing sessions after successful recovery.
7. Return the user to login rather than silently creating a session.

**Required backend decisions:** Token hashing, expiry, one-time use,
rate-limiting dimensions, notification channel, audit events, session
revocation, and recovery behavior for undeliverable email.

**Accessibility:** The accepted state is announced without implying account
existence. Password requirements are available before submission.

**Analytics:** `recovery_requested`, `recovery_completed`,
`recovery_failed`, without account identifier or token.

## 14. Complete onboarding and KYC

**Provisional entry:** `/onboarding`.

1. Explain why identity information is required before collecting it.
2. Separate profile completion, identity details, document capture, review, and
   status into resumable steps.
3. Validate document requirements before upload and preserve progress safely.
4. Show explicit `not_started`, `in_progress`, `submitted`, `needs_action`,
   `approved`, and `rejected` states only if the backend adopts them.
5. Provide a recovery action for upload failure and a human-support path for
   review problems.

**Required backend decisions:** Jurisdiction, provider, lawful basis,
verification status model, document-upload boundary, retention, deletion,
manual review, retry policy, and which financial capabilities require KYC.

**Security checkpoint:** Identity documents must never pass through analytics,
client logs, local storage, or an unapproved application server.

**Accessibility:** Camera capture always has a file-upload alternative.
Instructions do not rely on example imagery alone, and review status is
announced as text.

**Analytics:** Step-level completion and generic failure category only.

## 15. Enable MFA or step-up security

**Provisional entry:** `/app/security`.

1. Reauthenticate before enrollment.
2. Explain the chosen factor, recovery method, and lockout consequences.
3. Enroll the factor and require a successful challenge before activation.
4. Present recovery codes once with safe download and print guidance.
5. Require step-up authentication for future high-risk actions according to a
   backend policy, not frontend judgment.
6. Reauthenticate and audit factor removal.

**Required backend decisions:** Supported factors, challenge lifetime,
attempt limits, remembered-device policy, recovery codes, factor replacement,
step-up token scope, and session revocation after security changes.

**Accessibility:** One-time-code fields support paste, password managers, and a
single logically labelled value. Countdown timers never prevent completion by
screen-reader or motor-impaired users.

**Analytics:** Enrollment and challenge outcome only; never factor secrets,
codes, phone numbers, or recovery codes.

## 16. Manage active device sessions

**Provisional entry:** `/app/security/sessions`.

1. List recognizable sessions using bounded device, approximate location, and
   last-active information.
2. Clearly mark the current session.
3. Revoke one non-current session with confirmation.
4. Revoke all sessions with a stronger warning that includes the current
   device.
5. Refresh the list only after the backend confirms revocation.

**Current fallback:** Offer only `Logout all`, because listing and individual
revocation have no customer HTTP contract.

**Required backend decisions:** Session-list response, privacy-safe location
data, stable session identifier, current-session marker, individual revocation,
audit trail, and suspicious-session notification.

**Accessibility:** Session cards use headings and descriptive revoke-button
labels. Confirmation focus begins on the consequence.

**Analytics:** Session count band and action outcome only; no device, IP,
location, user-agent, or session identifier.

## 17. Manage a virtual card

**Provisional entry:** `/app/cards`.

1. List cards using masked identity and explicit lifecycle status.
2. Open card detail without revealing sensitive values by default.
3. Require reauthentication or step-up before revealing card details.
4. Freeze or unfreeze only after a focused confirmation.
5. Treat provider-pending status separately from success or failure.
6. Immediately update controls only after backend confirmation.

**Required backend decisions:** Issuing provider, PCI boundary, card status
model, reveal token, step-up policy, freeze semantics, authorization holds,
spending controls, webhook delivery, and replacement flow.

**3D boundary:** A lightweight card presentation may enhance the confirmed card
state, but secure values remain semantic DOM content and the controls work
without WebGL.

**Accessibility:** Masked card identity is readable without grouping ambiguity.
Freeze state uses text and status semantics. Any visual card has a complete
non-visual equivalent.

**Analytics:** Card lifecycle action and generic outcome only; never card
number, security code, expiry, token, provider ID, or cardholder data.

## Still blocked without detailed flows

- Phone verification, until its role relative to email and MFA is decided.
- User notification inbox, until feed and read-state contracts exist.
- External-bank transfers, until payment rails and settlement semantics exist.
