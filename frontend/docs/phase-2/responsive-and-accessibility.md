# Responsive and Accessibility Rules

## Accessibility target

WCAG 2.2 AA is the minimum release target. Accessibility is a component and
flow requirement, not a final audit task.

## Layout behavior

### Below 480 pixels

- Single-column content.
- 16-pixel page gutters.
- Stable bottom navigation outside focused tasks.
- Full-width form controls and primary actions.
- Account and transaction data use semantic lists.
- Dialogs become full-height sheets only when focus, keyboard, and back-button
  behavior remain predictable.
- Decorative authentication and marketing visuals are removed first.

### 480 to 767 pixels

- 20-pixel gutters where space permits.
- Two compact fields may share a row only when each remains at least 160 pixels
  wide and the grouping is semantically clear.
- Account summaries remain one per row.
- Filter controls may open in a labelled drawer.

### 768 to 1023 pixels

- Eight-column grid.
- Mobile bottom navigation may remain; do not introduce a cramped sidebar.
- Transfer review may place summary beneath form rather than beside it.
- Tables appear only when their columns fit without hiding critical values.

### 1024 pixels and above

- Persistent sidebar.
- Twelve-column content grid.
- Transfer form and review summary may use a 7/5 split.
- Account detail may use an 8/4 split for activity and account actions.
- Maximum content widths prevent excessively long rows and scanning distance.

### 1280 pixels and above

- Increase whitespace before increasing component size.
- Dashboard density remains stable.
- Marketing may use more expressive asymmetry.
- The primary balance does not continue scaling with viewport width.

## Reflow and zoom

- Core tasks work at 320 CSS pixels without two-dimensional scrolling.
- Content remains usable at 200 percent browser zoom.
- Text-spacing overrides do not clip labels, values, or controls.
- Sticky headers and action regions never obscure focused content.
- Browser and operating-system font scaling are respected.

## Semantic structure

- Each page has one descriptive `h1`.
- Headings do not skip levels for visual size.
- Use `main`, `nav`, `header`, `footer`, and `aside` landmarks correctly.
- Lists, tables, definition lists, fieldsets, and buttons retain native
  semantics.
- Clickable containers do not replace links or buttons.
- Transaction review uses a definition list, not a visual grid of generic
  `div` elements.

## Keyboard navigation

- All functionality works without a pointer.
- Focus order follows reading and visual order.
- Skip link targets the main page heading or content region.
- Menus support expected arrow, escape, home, and end behavior through tested
  accessible primitives.
- No positive `tabindex`.
- Dragging always has a non-drag alternative.
- Keyboard users can reach sticky controls without crossing hidden overlays.

## Focus

The focus indicator:

- Is at least 2 CSS pixels in its significant area.
- Uses a high-contrast outer ring plus optional offset.
- Is visible on paper, white, evergreen, warning, and danger surfaces.
- Is not removed on mouse interaction if the browser determines it is needed.
- Remains visible in Windows forced-colors mode.

Focus is never represented only by shadow softness or a subtle color shift.

## Forms and errors

- Labels remain visible after entry.
- Instructions precede the control when needed to complete it.
- `aria-describedby` associates hints and errors.
- Error summaries link to invalid fields.
- Focus moves to the summary after a failed submit with multiple errors.
- Valid fields are not announced repeatedly.
- Required and optional states are written in text.
- Autocomplete tokens are used correctly for identity and password fields.
- One-time codes support paste and password-manager workflows.

## Status announcements

Use restrained live regions:

| Event                   | Behavior                                                        |
| ----------------------- | --------------------------------------------------------------- |
| Validation after submit | Assertive summary, field messages available                     |
| Loading complete        | Polite only when context is not otherwise clear                 |
| Recipient resolved      | Polite announcement with masked identity                        |
| Transfer submitting     | Polite `Sending transfer` status                                |
| Transfer result         | Focus result heading; do not duplicate full page in live region |
| Copy complete           | Brief polite confirmation                                       |
| Offline/online change   | Polite persistent banner                                        |
| Session expired         | Accessible dialog with focused heading                          |

Live regions are present before messages are inserted. Rapid status changes are
debounced to avoid repeated announcements.

## Tables and financial data

- Data tables use captions or an associated heading.
- Column and row headers are marked correctly.
- Sort controls are buttons with `aria-sort`.
- Signed amounts have accessible direction text.
- Color-coded status always includes text.
- Mobile alternatives preserve the relationships between date, reference,
  status, and amount.
- Charts, when eventually justified, include the underlying values in a table
  or accessible summary.

## Dialogs and drawers

- Use a tested accessible primitive rather than hand-rolled focus traps.
- Dialog title and description are programmatically connected.
- Initial focus supports the task and avoids dangerous defaults.
- Escape closes only when dismissal is safe.
- Focus returns to the trigger.
- Background content becomes inert.
- Mobile drawers account for the virtual keyboard and safe-area insets.

## Touch and pointer

- Targets are at least 44 by 44 CSS pixels.
- Adjacent destructive and primary actions have sufficient separation.
- Hover information is also available through focus and touch.
- No action depends on precise dragging, multi-touch, or device orientation.
- Tooltips never contain required instructions.

## Contrast verification

Candidate foundation pairings:

| Foreground    | Background      | Contrast |
| ------------- | --------------- | -------- |
| White         | `evergreen-800` | 10.35:1  |
| White         | `danger-700`    | 7.06:1   |
| `ink-950`     | `paper-50`      | 15.36:1  |
| `ink-700`     | `paper-50`      | 8.70:1   |
| `ink-600`     | White           | 6.16:1   |
| `success-700` | `jade-100`      | 5.11:1   |
| `warning-700` | White           | 6.39:1   |
| `info-700`    | White           | 6.85:1   |

These checks validate only listed text pairs. Phase 3 must test all component
states, focus indicators, icons, borders, disabled controls, and charts.

## Motion and reduced motion

When `prefers-reduced-motion: reduce` is active:

- Page and layout transitions become immediate.
- Skeleton shimmer becomes static.
- Success animation is omitted.
- Smooth scrolling is disabled.
- Any future 3D scene becomes static or is not mounted.

No essential information is conveyed through movement. Animation does not
automatically replay when navigating back to a completed state.

## Screen-reader privacy

Visually masked information must also be masked in accessible names. Do not put
the full account number in an `aria-label` when the visual response is required
to remain masked.

Owned full account numbers use deliberate reveal/copy controls with clear
labels. Sensitive values are not placed in offscreen text as a shortcut.

## Color and display preferences

- Forced-colors mode retains borders, focus, status, and action hierarchy.
- Increased contrast preferences receive stronger boundaries where supported.
- Color-scheme metadata remains light for the initial release.
- A future dark theme requires a separate semantic palette and complete
  contrast review; it is not an inversion filter.

## Accessibility test matrix

Every core component is reviewed with:

- Keyboard only.
- VoiceOver with Safari.
- NVDA with Firefox or Chrome.
- 200 percent zoom.
- 320-pixel reflow.
- Reduced motion.
- Forced colors.
- Touch on a mid-range Android viewport.
- Slow network and delayed status announcements.

Critical transfer, verification, account closure, password change, and logout
flows require manual assistive-technology testing in addition to automation.
