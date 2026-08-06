# Navigation and System States

## Desktop application navigation

The primary sidebar order is:

1. Overview
2. Accounts
3. Transfer
4. Transactions
5. Beneficiaries
6. Cards, hidden until supported
7. Notifications, hidden until supported

Profile, security, settings, help, and sign out belong in the user menu and a
small secondary sidebar region. Transfer is visually prominent but must not
compete with the balance hierarchy.

The sidebar displays no full account numbers. Account switching belongs in page
content where currency, status, and current posted balance can be understood
together.

## Mobile navigation

Use a maximum of five bottom-navigation destinations:

1. Home
2. Accounts
3. Transfer
4. Activity
5. More

`More` opens beneficiaries, profile, security, settings, help, and sign out.
Cards and notifications appear only after their backend contracts exist.

The bottom navigation must not disappear during ordinary page loading. It may
be replaced by a focused header during transfer review, account closure, and
other critical confirmation steps.

## Breadcrumbs and page hierarchy

- Do not show breadcrumbs on the dashboard root or small mobile screens.
- Desktop pages deeper than one level show semantic breadcrumbs.
- Breadcrumb labels use human-readable values, never internal IDs.
- Account breadcrumbs use currency and masked account number.
- Transaction breadcrumbs use the public transaction reference.
- Breadcrumb links never bypass route guards.

Examples:

```text
Accounts / USD ******1756 / Statement
Transactions / TXN-ABC123
Beneficiaries / Add beneficiary
```

## Back navigation

- A browser back action retains safe form input until a financial operation has
  been submitted.
- After submission, browser back must not resubmit a transfer.
- Transfer review provides an explicit `Edit transfer` action.
- Successful transfer screens return to account detail or start a new transfer.
- Entry from a deep link falls back to the nearest safe parent rather than
  depending on browser history.
- Closing a modal restores focus to the element that opened it.

## Loading strategy

- Restore authentication before rendering protected financial data.
- Show structure-preserving skeletons, not fake values.
- Never display `0` while a balance request is unresolved.
- Load account identity and current posted balance together.
- Disable transfer submission while the mutation is in flight.
- Announce meaningful status changes through an `aria-live` region.
- The signup and login forms become usable independently of any future 3D scene.

## Empty states

Every empty state must explain why it is empty and provide one safe next action:

| Context               | Message intent                         | Primary action                      |
| --------------------- | -------------------------------------- | ----------------------------------- |
| No accounts           | Explain that accounts start at zero    | Create account                      |
| No transactions       | Confirm that no posted activity exists | Make transfer, if an account exists |
| No beneficiaries      | Explain saved recipients               | Add beneficiary                     |
| Filter has no results | Keep current data intact               | Clear filters                       |
| Closed account        | Preserve history and identity          | View statement                      |

Empty states must not use fake transactions, fake balances, or decorative
charts that could be mistaken for financial data.

## Error strategy

Errors are categorized by recovery behavior:

| Category                 | Presentation                                  | Recovery                            |
| ------------------------ | --------------------------------------------- | ----------------------------------- |
| Validation               | Inline beside the relevant field              | Correct input                       |
| Business rule            | Inline summary plus affected field or account | Change transfer intent              |
| Authentication           | Session-expired dialog or redirect            | Sign in again                       |
| Authorization/not found  | Neutral page state                            | Return to safe parent               |
| Rate limit               | Inline message with retry time when available | Wait and retry                      |
| Dependency outage        | Page or section error                         | Retry without losing safe input     |
| Unknown transfer outcome | Dedicated uncertain state                     | Retry with the same idempotency key |

Raw backend errors, PostgreSQL details, stack traces, recipient existence, and
security-control internals are never displayed.

## Offline and connection loss

- A persistent but unobtrusive banner announces offline status.
- Cached financial data is visibly marked with its last successful refresh
  time.
- Transfer submission, beneficiary creation, profile updates, and account
  closure are disabled while offline.
- Draft narration and non-sensitive form choices may remain in memory.
- Account numbers, tokens, passwords, and complete transfer payloads are not
  persisted to local storage.
- If connectivity fails after transfer submission, retain the same
  idempotency key and present an uncertain outcome until replay confirms the
  original result.

## Session expiration

- Expiration during passive browsing opens a clear dialog and preserves only a
  safe same-origin return route.
- Expiration before a financial mutation prevents submission and requires a
  fresh login.
- Expiration after submission must not label the transfer failed. Restore the
  session, then resolve the result using the transaction response or an
  idempotent replay.
- Signing out clears in-memory user data and query caches before navigation.
- `Logout all` warns that the current session will also end.

## Permission and lifecycle states

- Pending users can view and update profile information, inspect email status,
  resend verification, and sign out.
- Pending users cannot see enabled financial actions.
- Frozen accounts show a clear restriction: they can receive funds but cannot
  send.
- Closed accounts remain visible in history-oriented views but cannot be
  selected as transfer senders or recipients.
- Foreign-owned and missing account UUIDs share the same not-found state.
- Disabled registrations are directed to the recovery guidance returned by the
  backend and are not described as verified customers.

## Accessibility requirements

- Use landmarks, native controls, a logical heading hierarchy, and a skip link.
- Keyboard focus order follows visual order at every breakpoint.
- Visible focus indicators meet WCAG 2.2 AA and are never replaced by color
  alone.
- Minimum touch target is 44 by 44 CSS pixels.
- Dialogs trap focus only while open and restore it on close.
- Tables have proper headers and a meaningful small-screen alternative.
- Financial status uses text plus iconography, not color alone.
- Reduced-motion preferences remove nonessential transitions.
- All future 3D content has an equivalent static image or no-content fallback.
