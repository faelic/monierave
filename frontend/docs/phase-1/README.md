# Phase 1: Information Architecture and User Flows

## Status

Ready for product approval.

This phase defines how Monierave is organized and how users move through its
critical journeys. It does not create application pages, visual design tokens,
components, Three.js scenes, or a Next.js project.

## Phase outcome

Phase 1 establishes:

- A complete public and authenticated route model.
- Desktop and mobile navigation behavior.
- Breadcrumb, deep-link, back-navigation, and route-guard rules.
- Standard loading, empty, error, offline, restricted, and expired-session
  behavior.
- Critical user flows with validation, security, recovery, analytics, and
  accessibility requirements.
- A contract matrix separating confirmed backend functionality from future
  product scope.

## Documents

- [Route map](./route-map.md)
- [Navigation and system states](./navigation-and-states.md)
- [Critical user flows](./critical-user-flows.md)
- [Backend contract matrix](./backend-contract-matrix.md)
- [Decisions required](./decisions-required.md)

## Product boundaries

### Buildable with the current backend

- Registration and email verification.
- Login, access-token renewal, logout, and logout-all.
- Current-user profile restoration and profile updates.
- Account creation, listing, detail, statement, and closure.
- Recipient resolution by 10-digit account number.
- Beneficiary management.
- Internal, same-currency, zero-fee transfers.
- Transaction history and transaction detail.

### Not yet buildable end to end

- Password recovery.
- Phone verification.
- KYC and identity-document review.
- MFA and step-up authorization.
- Session/device listing and individual session revocation.
- Virtual cards.
- User-facing notification inbox.
- External-bank transfers, payment rails, FX, transfer quotes, holds, and card
  authorizations.

These features remain visible in the long-term sitemap but are marked blocked.
They must not be presented as available in the initial application.

## Binding terminology

- `Current posted balance`: the balance exposed by the backend. The interface
  must not label it available balance because holds are not modeled.
- `Account ID`: the public UUID used in owned resource URLs.
- `Account number`: the 10-digit customer-facing routing identifier.
- `Internal transfer`: a transfer between Monierave accounts in the same
  currency. Its current fee is zero.
- `Pending registration`: a signed-in user who has not completed email
  verification and cannot access financial features.

## Approval gate

Phase 1 is approved when:

- The MVP route map contains no unsupported product claims.
- Public, pending-registration, verified, and expired-session boundaries are
  unambiguous.
- The transfer flow preserves recipient privacy, idempotency, and truthful
  posted-balance semantics.
- Every critical flow has a recovery path and accessible status behavior.
- Desktop and mobile navigation expose the same essential capabilities.
- Future routes are clearly blocked rather than silently treated as confirmed.

Phase 2, visual direction and design system, must not begin until this gate is
explicitly approved.
