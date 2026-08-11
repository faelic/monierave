# Monierave Implementation Instructions

Monierave is a Go banking-learning backend with a Next.js frontend. Preserve
the security and financial contracts described in `README.md`, the database
schema, and the frontend phase documentation.

The production marketing hero uses the lightweight rotating celestial globe in
`frontend/src/features/rotating-globe-hero`. Its approved boundaries are:

- inline SVG and CSS for the globe geometry and rotation
- DOM navigation, copy, and calls to action
- deterministic, restrained motion
- a meaningful static state for reduced motion
- no WebGL or historical experimental hero dependencies

The retired wallet, card/ribbon, and pipeline experiments have been removed.
Do not restore them or add replacement design references without an explicit
new direction from the user.

Before deployment, complete `docs/DEPLOYMENT-CHECKLIST.md`.
