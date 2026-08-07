# Design Foundations

## Typography

### Recommended families

| Role              | Family                                                                  | Use                                              |
| ----------------- | ----------------------------------------------------------------------- | ------------------------------------------------ |
| Editorial display | Fraunces variable                                                       | Marketing headlines and rare orientation moments |
| Interface         | Source Sans 3 variable                                                  | Forms, navigation, tables, balances, body copy   |
| Fallback          | A metrically compatible bundled fallback selected during implementation | Prevent disruptive layout shift                  |

Exact font package versions, subsets, and licensing must be verified and pinned
during Phase 3. The application must remain usable before web fonts load.

Fraunces is not used for amounts, controls, transaction rows, or security
instructions. Source Sans 3 supports dense information and clear character
distinction. Financial numbers use:

```css
font-variant-numeric: tabular-nums lining-nums;
```

### Type scale

| Token        | Size / line height | Weight | Typical use                      |
| ------------ | ------------------ | ------ | -------------------------------- |
| `display-xl` | 64 / 68            | 520    | Marketing hero, desktop only     |
| `display-lg` | 48 / 54            | 520    | Marketing section lead           |
| `heading-xl` | 36 / 42            | 650    | Major page orientation           |
| `heading-lg` | 28 / 34            | 650    | Page heading                     |
| `heading-md` | 22 / 28            | 650    | Section heading                  |
| `heading-sm` | 18 / 24            | 650    | Card or dialog heading           |
| `body-lg`    | 18 / 28            | 400    | Introductory copy                |
| `body-md`    | 16 / 24            | 400    | Default body and form controls   |
| `body-sm`    | 14 / 20            | 400    | Supporting information           |
| `label`      | 14 / 18            | 600    | Labels and compact navigation    |
| `caption`    | 12 / 16            | 500    | Timestamps and tertiary metadata |
| `amount-xl`  | 40 / 44            | 650    | Primary account balance          |
| `amount-md`  | 20 / 26            | 650    | Transaction and review amounts   |

Mobile marketing display sizes step down one level. Body text never drops below
14 pixels; form controls remain at least 16 pixels to prevent mobile zoom.

## Spacing

Use a 4-pixel base:

| Token      | Value |
| ---------- | ----- |
| `space-0`  | 0     |
| `space-1`  | 4     |
| `space-2`  | 8     |
| `space-3`  | 12    |
| `space-4`  | 16    |
| `space-5`  | 20    |
| `space-6`  | 24    |
| `space-8`  | 32    |
| `space-10` | 40    |
| `space-12` | 48    |
| `space-16` | 64    |
| `space-20` | 80    |
| `space-24` | 96    |

Component internals primarily use 8, 12, 16, and 24. Page rhythm primarily
uses 24, 32, 48, and 64. Arbitrary spacing requires a documented optical
reason.

## Grid and content width

| Context        | Columns | Gutter | Maximum width |
| -------------- | ------- | ------ | ------------- |
| Small mobile   | 4       | 16     | Fluid         |
| Large mobile   | 4       | 20     | Fluid         |
| Tablet         | 8       | 24     | Fluid         |
| Desktop        | 12      | 24     | 1280          |
| Wide marketing | 12      | 32     | 1440          |

Banking page content should rarely exceed 1200 pixels even when the application
shell is wider. Reading content uses a 68-character maximum measure. Forms use
480 to 560 pixels unless a review panel shares the row.

## Breakpoints

Breakpoints follow layout pressure rather than named devices:

| Token | Width  | Intent                                       |
| ----- | ------ | -------------------------------------------- |
| `xs`  | 480px  | Large-phone refinements                      |
| `sm`  | 640px  | Form and list breathing room                 |
| `md`  | 768px  | Tablet navigation and two-column eligibility |
| `lg`  | 1024px | Persistent application sidebar               |
| `xl`  | 1280px | Full dashboard grid                          |
| `2xl` | 1440px | Marketing composition only                   |

Components must work between breakpoints, not only at them.

## Color system

### Core palette

| Token           | Value     | Role                              |
| --------------- | --------- | --------------------------------- |
| `ink-950`       | `#14211D` | Primary text                      |
| `ink-700`       | `#394A43` | Secondary text                    |
| `ink-600`       | `#55655E` | Muted text where contrast permits |
| `evergreen-900` | `#0E332C` | Brand anchor and dark navigation  |
| `evergreen-800` | `#16483D` | Primary action                    |
| `evergreen-700` | `#1C5B4C` | Hover and selected accents        |
| `jade-500`      | `#24A779` | Controlled accent and progress    |
| `jade-100`      | `#D8F1E7` | Positive tint                     |
| `paper-50`      | `#F8F6F0` | Warm page background              |
| `paper-100`     | `#F0EDE4` | Secondary warm surface            |
| `white`         | `#FFFFFF` | Raised and form surfaces          |
| `line-200`      | `#D8DED9` | Default border and rule           |
| `line-300`      | `#C4CDC7` | Strong border                     |
| `success-700`   | `#15704D` | Success text and icon             |
| `warning-700`   | `#8A5200` | Warning text and icon             |
| `danger-700`    | `#A62A22` | Error and destructive action      |
| `info-700`      | `#17627A` | Informational status              |

These are candidate values, not implementation-ready tokens. Phase 3 must
programmatically verify every text/background and component-state pairing
against WCAG 2.2 AA. Failed pairings are adjusted before coding components.

### Role rules

- Primary action uses `evergreen-800` with white text.
- Jade is an accent, not a default text color.
- Danger is reserved for actual destructive or failed states.
- Warning does not represent pending; pending uses neutral or info styling.
- Large surface areas use paper and white, not saturated green.
- Dark mode is not part of the initial release. It must not be created by
  mechanically inverting these colors.

## Borders and dividers

- Default rule: 1 pixel `line-200`.
- Strong selected or focused boundary: 2 pixels.
- Transaction rows favor horizontal rules over boxed cards.
- Dashed borders are reserved for optional upload/drop regions.
- Validation does not change layout width when a border becomes stronger.

## Radius

| Token         | Value | Use                                     |
| ------------- | ----- | --------------------------------------- |
| `radius-xs`   | 4px   | Tags and compact controls               |
| `radius-sm`   | 8px   | Inputs, buttons, menu items             |
| `radius-md`   | 12px  | Cards, dialogs, drawers                 |
| `radius-lg`   | 16px  | Marketing media and rare feature panels |
| `radius-pill` | 999px | Status badges only                      |

Do not use large pill-shaped containers for forms, account cards, tables, or
navigation.

## Elevation

Elevation communicates layering, not importance:

| Level | Use                         |
| ----- | --------------------------- |
| `0`   | Page content and cards      |
| `1`   | Sticky navigation and menus |
| `2`   | Popovers and drawers        |
| `3`   | Dialogs                     |

Cards at level 0 use borders or surface contrast rather than shadows. Shadows
are neutral, soft, and low-opacity. No colored glows.

## Iconography

- Use one outlined icon family with a consistent 1.75 to 2 pixel stroke.
- Default sizes are 16, 20, and 24 pixels.
- Icons supplement labels; critical actions are not icon-only.
- Directional transaction icons include readable incoming/outgoing text.
- Brand illustrations and interface icons remain separate systems.
- Exact package selection belongs to Phase 3 after bundle and coverage review.

## Motion

| Token         | Duration      | Use                                |
| ------------- | ------------- | ---------------------------------- |
| `instant`     | 0ms           | Reduced motion and immediate state |
| `fast`        | 120ms         | Hover, focus, compact disclosure   |
| `standard`    | 180ms         | Menus and inline state changes     |
| `deliberate`  | 260ms         | Drawer and page-region transitions |
| `celebratory` | 500ms maximum | Confirmed success, once            |

Recommended easing:

```text
enter: cubic-bezier(0.16, 1, 0.3, 1)
exit:  cubic-bezier(0.4, 0, 1, 1)
```

Avoid bounce, overshoot, perpetual pulsing, count-up balances, and motion that
implies success before the backend confirms it.
