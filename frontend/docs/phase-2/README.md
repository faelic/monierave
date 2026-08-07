# Phase 2: Visual Direction and Design System

## Status

Ready for visual-system approval.

This phase defines Monierave's visual language and interaction standards before
frontend scaffolding. It does not create React components, Tailwind
configuration, Storybook stories, pages, illustrations, or 3D scenes.

## Design proposition

Monierave should feel like a calm financial workspace rather than a promotional
fintech toy. The interface uses warm editorial restraint, precise financial
hierarchy, compact controls, and a deep evergreen identity. It avoids the
purple gradients, excessive glass, oversized cards, decorative charts, and
continuous motion common in interchangeable fintech products.

The visual system has two coordinated expressions:

- `Editorial trust`: warmer typography, generous composition, and selective
  imagery for marketing and authentication.
- `Operational clarity`: denser information, quiet surfaces, and predictable
  controls for banking and security work.

Both expressions share color roles, type metrics, spacing, focus behavior, and
content rules.

## Documents

- [Brand and visual direction](./brand-and-visual-direction.md)
- [Foundations](./foundations.md)
- [Components and patterns](./components-and-patterns.md)
- [Responsive and accessibility rules](./responsive-and-accessibility.md)
- [Content and financial-data rules](./content-and-financial-data.md)
- [Quality gates and decisions](./quality-gates.md)

## Non-negotiable principles

1. Financial truth outranks decoration.
2. Current posted balance is never described as available balance.
3. Status is communicated through text, iconography, and structure, not color
   alone.
4. Irreversible actions receive focused confirmation, not generic modal copy.
5. Loading states never display invented balances or transactions.
6. Motion clarifies change but never competes with transfer review or security
   decisions.
7. Core banking remains complete when animation, custom fonts, imagery, or
   WebGL is unavailable.
8. Compact does not mean cramped; generous does not mean oversized.

## Phase boundary

Phase 3 may translate these specifications into tokens, primitives, Storybook,
and application foundations only after:

- Color contrast is verified.
- Typography and font-loading strategy are approved.
- Core component anatomy and states are approved.
- Mobile density and critical-confirmation patterns are approved.
- The unresolved beneficiary-transfer contract remains visible in planning.

No polished Three.js production begins in Phase 2 or Phase 3.
