# Route Map

## Route groups

The eventual Next.js application should use conceptual route groups without
placing group names in public URLs:

```text
(marketing)     Public product and trust content
(auth)          Registration, verification, and authentication
(onboarding)    Profile and future identity setup
(app)           Verified customer banking experience
(legal)         Legal and privacy documents
(system)        Global error, maintenance, and connection states
```

`/app` is used below to make the authenticated boundary explicit. The final
brand URL may omit that prefix, but that decision must be consistent across the
entire product.

## Public and authentication routes

| Route                  | Purpose                                                             | Access                     | Primary action          | Risk     | 3D                          | Delivery               |
| ---------------------- | ------------------------------------------------------------------- | -------------------------- | ----------------------- | -------- | --------------------------- | ---------------------- |
| `/`                    | Explain Monierave and direct users to registration                  | Public                     | Create account          | Low      | Optional Tier A             | Later marketing phase  |
| `/security`            | Explain security practices without compliance claims                | Public                     | Review safeguards       | Medium   | None                        | Later marketing phase  |
| `/help`                | Entry point for help content                                        | Public                     | Find assistance         | Low      | None                        | Later support phase    |
| `/legal/privacy`       | Privacy notice                                                      | Public                     | Read policy             | Medium   | None                        | Legal content required |
| `/legal/terms`         | Product terms                                                       | Public                     | Read terms              | Medium   | None                        | Legal content required |
| `/login`               | Authenticate an existing user                                       | Anonymous only             | Sign in                 | High     | Optional Tier A beside form | MVP                    |
| `/signup`              | Create a pending registration                                       | Anonymous only             | Register                | High     | Optional Tier A beside form | MVP                    |
| `/signup/check-email`  | Explain the next step after registration without assuming a session | Public                     | Open verification email | Medium   | Static fallback only        | MVP                    |
| `/verify-email`        | Validate and confirm an emailed token                               | Public token flow          | Confirm email           | High     | None                        | MVP                    |
| `/verification-needed` | Explain restricted pending status                                   | Authenticated pending user | Resend or update email  | Medium   | Static illustration only    | MVP                    |
| `/forgot-password`     | Start account recovery                                              | Anonymous only             | Request recovery        | Critical | None                        | Blocked by backend     |
| `/reset-password`      | Complete recovery using a one-time token                            | Public token flow          | Set password            | Critical | None                        | Blocked by backend     |
| `/maintenance`         | Planned service interruption                                        | Public                     | Retry later             | Medium   | None                        | Later operations phase |

Authenticated users who visit `/login` or `/signup` should be redirected
according to their state: verified users to `/app`, pending users to
`/verification-needed`. A successful signup does not create a session, so it
routes to the public `/signup/check-email` state rather than an authenticated
page.

## Banking routes

| Route                                 | Purpose                                            | Access               | Primary action                      | Risk     | 3D                              | Delivery                |
| ------------------------------------- | -------------------------------------------------- | -------------------- | ----------------------------------- | -------- | ------------------------------- | ----------------------- |
| `/app`                                | Banking overview using current posted balances     | Verified             | Review position and recent activity | High     | None                            | MVP                     |
| `/app/accounts`                       | List owned accounts                                | Verified             | Select or create account            | High     | None                            | MVP                     |
| `/app/accounts/new`                   | Create a zero-balance currency account             | Verified             | Create account                      | High     | None                            | MVP                     |
| `/app/accounts/[accountId]`           | Account detail and recent activity                 | Verified owner       | Review account                      | High     | None                            | MVP                     |
| `/app/accounts/[accountId]/statement` | Filtered account statement                         | Verified owner       | Review or export later              | High     | None                            | MVP                     |
| `/app/accounts/[accountId]/close`     | Explain and confirm closure                        | Verified owner       | Close zero-balance account          | Critical | None                            | MVP                     |
| `/app/transactions`                   | Cross-account transaction history                  | Verified             | Search and filter activity          | High     | None                            | Partial backend support |
| `/app/transactions/[reference]`       | Transaction receipt and status                     | Verified participant | Review transaction                  | High     | None                            | MVP                     |
| `/app/transfers/new`                  | Enter sender, recipient, amount, and narration     | Verified             | Prepare transfer                    | Critical | None                            | MVP                     |
| `/app/transfers/review`               | Review immutable transfer intent before submit     | Verified             | Confirm transfer                    | Critical | None                            | MVP                     |
| `/app/transfers/[reference]`          | Show posted, uncertain, failed, or reversed result | Verified participant | Understand outcome                  | Critical | Short Tier C only after success | MVP                     |
| `/app/beneficiaries`                  | Manage saved recipients                            | Verified             | Select, rename, or remove           | High     | None                            | MVP                     |
| `/app/beneficiaries/new`              | Resolve and save a destination                     | Verified             | Add beneficiary                     | High     | None                            | MVP                     |

The current backend provides account-scoped history rather than a dedicated
cross-account list. `/app/transactions` is therefore partial: MVP may aggregate
already-loaded account histories or redirect users to select an account. It
must not imply a confirmed global-history API.

## Profile, security, and future routes

| Route                    | Purpose                                    | Access         | Primary action            | Risk     | 3D               | Delivery                  |
| ------------------------ | ------------------------------------------ | -------------- | ------------------------- | -------- | ---------------- | ------------------------- |
| `/app/profile`           | View personal and verification information | Authenticated  | Review profile            | High     | None             | MVP                       |
| `/app/profile/edit`      | Update name, email, or password            | Authenticated  | Save profile              | Critical | None             | MVP                       |
| `/app/security`          | Security overview                          | Authenticated  | Review controls           | Critical | None             | Partial                   |
| `/app/security/sessions` | Manage devices and active sessions         | Authenticated  | Revoke session            | Critical | None             | Blocked except logout-all |
| `/app/settings`          | Product preferences                        | Authenticated  | Update preferences        | Medium   | None             | Later                     |
| `/app/notifications`     | User notification inbox                    | Verified       | Review alerts             | High     | None             | Blocked by backend        |
| `/app/cards`             | List virtual cards                         | Verified       | Select card               | Critical | Optional Tier C  | Blocked by backend        |
| `/app/cards/[cardId]`    | Manage a virtual card                      | Verified owner | Freeze or reveal controls | Critical | Optional Tier C  | Blocked by backend        |
| `/onboarding`            | Resume onboarding                          | Authenticated  | Continue setup            | High     | Static or Tier B | Partial                   |
| `/onboarding/identity`   | Submit KYC information                     | Authenticated  | Verify identity           | Critical | None             | Blocked by backend        |
| `/onboarding/review`     | Review KYC status                          | Authenticated  | Resolve issue             | Critical | None             | Blocked by backend        |

## Route guard precedence

Guards must resolve in this order:

1. Restore the authenticated session using `GET /users/me`.
2. If authentication fails, preserve a safe same-origin return path and move to
   `/login`.
3. If the route is a financial route and the user is pending, move to
   `/verification-needed`.
4. If the route references an owned resource, allow the backend to enforce
   ownership and render the same not-found state for missing and foreign data.
5. If a capability is not implemented, do not expose its navigation entry.

## Deep links

- Deep links must survive a full page refresh.
- A safe return path may contain only an application-relative path. Absolute,
  protocol-relative, and cross-origin redirects are rejected.
- Filters and pagination belong in search parameters when they are shareable.
- Sensitive tokens must be removed from the visible URL after they are
  exchanged or submitted.
- A deep link to an unavailable or foreign resource renders a neutral not-found
  state without revealing whether the resource exists.
