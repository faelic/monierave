# Monierave - Start Here

Read `AGENTS.md` first.

## Current frontend direction

The production landing page uses a rotating celestial money globe built with
inline SVG and CSS. Navigation, content, controls, and accessibility remain in
the DOM. The globe freezes into a complete static composition when reduced
motion is requested.

Authentication uses a CSS-based visual atmosphere and does not load WebGL.
Authenticated banking screens prioritize clarity, security state, and truthful
financial terminology over decorative motion.

## Working rules

- Keep product behavior aligned with backend authorization and lifecycle rules.
- Preserve responsive, keyboard, forced-color, and reduced-motion support.
- Keep access credentials in memory and refresh credentials in secure cookies.
- Do not restore removed design experiments or reference media.
- Complete `docs/DEPLOYMENT-CHECKLIST.md` before deployment.
