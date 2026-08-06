# Components and Interaction Patterns

## Component principles

- Components express semantic roles, not page-specific decoration.
- Native HTML behavior is preferred before headless primitives.
- Every interactive component defines default, hover, active, focus-visible,
  disabled, loading, error, and reduced-motion behavior where applicable.
- Disabled controls must have an adjacent explanation when the reason is not
  obvious.
- Financial mutation components never use optimistic success.

## Buttons

### Variants

| Variant   | Use                                      | Restrictions                            |
| --------- | ---------------------------------------- | --------------------------------------- |
| Primary   | One main forward action in a task region | Maximum one per focused region          |
| Secondary | Safe alternative or supporting action    | May sit beside primary                  |
| Quiet     | Low-emphasis navigation or disclosure    | Not for irreversible actions            |
| Danger    | Confirmed destructive action             | Only inside a clear consequence context |
| Link      | Navigation embedded in text              | Must look interactive without hover     |

Buttons use 8-pixel radius, visible labels, and minimum 44-pixel touch height.
Loading preserves width, disables repeat activation, and retains the action
label for assistive technology. A spinner alone is insufficient.

`Copy` actions may use an icon plus label. Their confirmation is temporary text
and an `aria-live` announcement, not a toast that exposes copied data.

## Form controls

### Field anatomy

1. Persistent label.
2. Optional instruction or format hint.
3. Input, select, textarea, or composed control.
4. Error or supporting message.
5. Optional character or constraint status.

Placeholders are examples, never labels. Required state is stated in text.
Errors identify both the problem and the corrective action.

### Amount input

- Currency is fixed by or explicitly selected with the sender account.
- The visible control accepts a familiar major-unit format.
- Parsing is locale-aware but produces an exact integer minor-unit value.
- Floating-point arithmetic is prohibited.
- Grouping separators are presentation only.
- The review step shows the canonical amount and currency again.
- Invalid precision, negative values, zero, and overflow are rejected before
  submission.

### Account-number input

- Accept digits only after normalization.
- Preserve leading zeroes.
- Use ten visible character positions without splitting into inaccessible
  one-character inputs.
- Allow paste.
- Never autocomplete from unrelated personal fields.
- Full numbers are cleared from persistent state after the flow ends.

### Password input

- Provide reveal/hide control with explicit state.
- Show 8-character minimum and 72-byte maximum before submit.
- Do not impose composition theater.
- Clear password values after submission failure that crosses a network or
  authentication boundary.

### Select and combobox

Use a native select when search is unnecessary. Use an accessible combobox for
long searchable sets only. Account selection displays:

- Currency.
- Masked account number.
- Lifecycle status.
- Current posted balance, when already confirmed and appropriate.

Frozen and closed accounts remain understandable but cannot be selected as
senders.

## Status badge

Badge anatomy combines icon, text, and subtle tint. Supported product statuses
include:

- Active.
- Frozen.
- Closed.
- Pending.
- Posted.
- Failed.
- Reversed.
- Needs action.

Badges do not carry full explanatory burden. The surrounding content states the
effect, such as `Frozen - can receive, cannot send`.

## Alerts and inline notices

| Pattern              | Use                                             |
| -------------------- | ----------------------------------------------- |
| Inline field message | One input or selection                          |
| Section notice       | Affects one task region                         |
| Page banner          | Connection, maintenance, or broad account state |
| Blocking alert       | User cannot safely continue                     |

Notices have a semantic heading when they contain more than one sentence.
Dismissal is offered only when losing the message is safe.

## Toasts

Toasts acknowledge low-risk, already-completed actions such as copied text or
renamed beneficiary. They are not used as the sole confirmation for:

- Transfer outcome.
- Account closure.
- Password change.
- Session revocation.
- Verification failure.

Toasts pause on hover/focus, remain available long enough to read, and are
announced without stealing focus.

## Generic card

A card is used only when its content forms one object or decision boundary.
Card anatomy may include heading, supporting metadata, content, and a compact
action region. Rules:

- The entire card is clickable only when it has one navigation destination.
- Nested interactive controls require a non-clickable container.
- Heading levels follow page hierarchy rather than card styling.
- Cards do not nest inside cards.
- Border or surface contrast is preferred to elevation.
- Equal height is used only when it improves comparison, not as a default.

## Tooltip

Tooltips provide short supplementary definitions for unfamiliar terms. They:

- Are never required to complete a task.
- Open on hover and keyboard focus.
- Remain hoverable where needed.
- Close with escape.
- Do not contain interactive controls.
- Do not replace labels, error messages, or mobile instructions.

On touch layouts, required explanations appear inline or through a labelled
disclosure rather than hover-dependent UI.

## Menu and popover

Menus contain actions; popovers contain non-modal supporting content. Both use
tested accessible primitives and predictable keyboard behavior. Destructive
menu items are separated and labelled, but selecting one still opens the
appropriate confirmation rather than immediately mutating financial data.

## Notification item

The future notification inbox is backend-blocked, but its visual contract is:

1. Event category and readable heading.
2. Exact or relative time with exact detail available.
3. Read/unread state communicated beyond color.
4. Short privacy-safe summary.
5. One destination when the related resource is available.

Notifications never reveal full account numbers, balances, authentication
events beyond safe guidance, or sensitive card data on an unlocked screen.
Read state must not be implemented optimistically unless its future backend
contract supports retry-safe updates.

## Account summary

The account summary is an object card, not a decorative dashboard tile.

### Anatomy

1. Currency and account label.
2. Masked account number with optional copy/reveal policy for owned data.
3. Lifecycle status.
4. `Current posted balance` label.
5. Tabular amount.
6. Primary safe action and secondary detail link.

The status and currency remain visible when the amount is hidden for privacy.
Closed accounts use the same structure and retain statement access.

## Balance display

- Always include currency.
- Use tabular lining numbers.
- Keep minus sign and amount on one line.
- Do not animate from zero.
- Do not abbreviate primary balances to `1.2K`.
- If privacy masking is enabled later, mask all account balances consistently.
- Loading uses a shape skeleton without a fake currency value.
- Stale values show the last successful update time.

## Transaction row and table

### Desktop table columns

1. Date/time.
2. Counterparty or transaction type.
3. Narration or reference.
4. Status.
5. Signed amount.
6. Optional running balance in statements.

The row is one link to detail; nested buttons are avoided. Amount alignment is
right, text alignment follows reading direction, and status remains readable
without color.

### Mobile transaction item

Use a two-line semantic list:

- Counterparty/type and signed amount on the first line.
- Date, status, and reference/narration on the second.

Do not horizontally scroll the primary transaction list. Detailed statement
data may use a responsive table with an equivalent list or labelled cells.

## Filters and pagination

- Shareable filters live in URL search parameters.
- Active filters appear as removable text chips.
- `Clear all` is available when more than one filter is active.
- Applying filters does not erase previously rendered results until the new
  response is ready.
- Cursor pagination uses `Load more` or controlled next/previous behavior; it
  never fabricates page numbers.
- Account and beneficiary offset pagination may use page controls when needed.

## Recipient resolution

The resolution panel displays:

- Masked account name.
- Masked account number.
- Currency.
- `Can receive` state.
- A statement that lookup does not guarantee transfer success.

The success panel is visually neutral, not celebratory. A resolved identity is
not yet a completed financial action. Missing, malformed, and closed accounts
share one neutral not-found pattern.

## Beneficiary item

The item displays nickname first, then masked number, currency, and receiving
eligibility. Rename and remove live in an overflow menu with text labels.

Until the transfer-by-beneficiary backend contract exists, the item must not
show a misleading `Send` action after reload.

## Transfer stepper

Use three conceptual steps:

1. Details.
2. Review.
3. Result.

The stepper communicates location, not completion predicted by the client.
Mobile may replace the full step labels with `Step 2 of 3` plus the current
title.

### Details

Sender, recipient, amount, and narration are editable. Recipient resolution is
complete before review.

### Review

Use a definition list for:

- From account.
- Recipient.
- Amount.
- Fee, explicitly `0`.
- Total debit, equal to amount while fee is zero.
- Narration.

The confirm button states the action and amount where space permits. `Edit
transfer` is secondary.

### Result

Posted, uncertain, rejected, and reversed are separate result structures.
Result pages include transaction reference when confirmed. A success visual may
play once after the posted response and must not delay access to the receipt.

## Dialogs

Dialogs are used when context should remain visible and the task is short.
Full pages are used for multi-field or security-sensitive operations.

Dialog requirements:

- Semantic title and description.
- Initial focus chosen by task, never automatically on danger.
- Escape and explicit close for safe dismissals.
- Focus containment and restoration.
- Background inertness.
- No stacked dialogs.

## Destructive confirmation

Account closure, beneficiary removal, and logout-all use consequence-specific
copy. Account closure uses a page or large focused dialog containing:

- Masked account identity.
- Current posted balance.
- Permanent effect.
- Reason it may be unavailable.
- Neutral cancel action.
- Danger confirmation action.

Typed confirmation is not required unless risk analysis demonstrates value; it
often adds friction without proving intent.

## Navigation components

### Sidebar

- One active destination.
- Text labels remain visible at normal desktop widths.
- Collapsed icon-only mode is not an MVP requirement.
- Secondary user actions remain visually separate.

### Bottom navigation

- Maximum five destinations.
- Labels remain visible.
- Safe-area padding is included.
- Active state uses weight, icon treatment, and indicator rather than color
  alone.

### Breadcrumb

- Uses an ordered list and current-page semantics.
- Truncates middle segments before account identity or current page.
- Is hidden where it duplicates a mobile back affordance.

## Skeletons

Skeletons approximate final structure and use a quiet luminance shift. They:

- Stop under reduced motion.
- Never include fake digits.
- Preserve layout to minimize cumulative shift.
- Are replaced by explicit errors after failure.
- Do not persist indefinitely without status text.

## Progress indicators

- Determinate progress is used only when a meaningful total exists.
- Indeterminate progress includes text describing the active operation.
- Transfer submission does not use a fake percentage.
- Multi-step onboarding shows completed, current, and remaining steps without
  treating backend review as user-controlled progress.
- Progress animation stops under reduced motion.

## Empty states

An empty state contains:

1. Concise factual heading.
2. One sentence explaining the state.
3. One safe next action, when available.
4. Optional lightweight static illustration.

It never contains fabricated financial examples that resemble customer data.
