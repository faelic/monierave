# Phase 2 Quality Gates and Decisions

## Gate 1: Visual direction

Approve:

- `The Quiet Ledger` as the core concept.
- Warm paper surfaces with deep evergreen structure and controlled jade accent.
- Editorial expression for marketing/authentication and operational expression
  for banking.
- Open ruled layouts instead of pervasive floating cards.

Reject the gate if the direction feels decorative, crypto-oriented, overly
institutional, or indistinguishable from a generic gradient fintech template.

## Gate 2: Typography

Approve:

- Fraunces variable for limited editorial display use.
- Source Sans 3 variable for the complete product interface.
- Tabular lining figures for amounts and financial tables.
- A performance budget and fallback-metric review in Phase 3.

Required implementation proof:

- No layout break before fonts load.
- No clipped text at 200 percent zoom or increased text spacing.
- Currency, zero, uppercase `O`, one, lowercase `l`, and minus signs remain
  distinguishable.

## Gate 3: Color and states

Approve the semantic direction, not merely the palette swatches.

Required implementation proof:

- WCAG 2.2 AA contrast across every interactive state.
- Status meaning survives grayscale and forced colors.
- Focus remains visible on all surfaces.
- Disabled state remains readable and is not confused with loading.
- Warning, pending, and failure remain semantically distinct.

## Gate 4: Density and shape

Approve:

- 8-pixel input/button radius.
- 12-pixel card/dialog radius.
- Pills only for status badges.
- Borders and surface shifts before shadows.
- Compact financial rows with 44-pixel minimum interaction targets.

Reject oversized dashboard cards, excessive empty space, nested containers, or
mobile layouts that hide primary financial context below decoration.

## Gate 5: Financial components

The following component anatomy must be approved before page implementation:

- Account summary.
- Current posted balance.
- Transaction row and responsive list.
- Amount input.
- Account-number input.
- Recipient resolution panel.
- Beneficiary item.
- Transfer details, review, and result states.
- Account-closure confirmation.

Each must include loading, empty, error, offline, disabled, and accessible
behavior where relevant.

## Gate 6: Critical screen hierarchy

Prototype reviews in Phase 3 must demonstrate:

- Dashboard: balance, account status, and recent activity are understood in
  that order.
- Transfer: sender, recipient, amount, fee, total, and outcome are never
  ambiguous.
- Verification: restriction and next action are immediately clear.
- Security confirmation: consequence is understood before the danger action.
- Mobile: primary action remains reachable without hiding important content.

## Gate 7: Responsive and accessibility

Required proof:

- Core flows work from 320 pixels through wide desktop.
- Keyboard and screen-reader behavior is documented in Storybook.
- Touch targets are at least 44 pixels.
- Focus is not obscured by sticky regions.
- Motion is optional.
- Web fonts and future 3D are non-blocking enhancements.

## Decisions adopted from Phase 1

Proceeding to Phase 2 uses these working assumptions:

1. `/app` remains the authenticated banking prefix.
2. Pending users receive a focused verification layout, not the banking shell.
3. Unsupported KYC, card, notification, MFA, and device-management navigation
   remains hidden.
4. MVP transaction history requires account selection.
5. Saved-beneficiary transfer remains blocked until the backend supports a
   beneficiary identifier or the user re-enters the full destination number.

Assumption 5 must be resolved before implementing beneficiary selection inside
the transfer flow.

## Decisions requiring approval

### Typography licensing and delivery

Confirm that Fraunces and Source Sans 3 licensing and delivery fit the project.
Recommended delivery is framework-managed, self-hosted font files with only
required subsets and styles.

### Owned account-number display

Recommendation: show full account number only on owned account detail with
explicit copy control; use masking in lists, breadcrumbs, transfer review, and
all recipient contexts.

### Initial dark mode

Recommendation: do not include dark mode in the first release. Build semantic
tokens so a deliberately designed dark palette can be added later.

### Success animation

Recommendation: allow one restrained, sub-500-millisecond DOM animation after a
confirmed posted transfer. 3D success treatment remains a later performance-
gated enhancement.

## Phase 2 definition of done

Phase 2 is complete when:

- Brand personality and visual concept are documented.
- Typography, scale, spacing, grid, breakpoints, color roles, surfaces,
  borders, radius, elevation, icons, and motion are specified.
- Core component anatomy and state behavior are specified.
- Marketing, authentication, dashboard, transactional, security, and mobile
  expressions are distinct but coherent.
- Financial terminology and amount formatting are binding.
- WCAG 2.2 AA behavior is integrated into the design specification.
- Candidate core text colors pass contrast checks.
- Decisions and unresolved backend dependencies are visible.
- No frontend scaffold, page code, or 3D production has begun.

Phase 3, frontend foundation and repository setup, begins only after explicit
approval.
