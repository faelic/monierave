# Monierave Web

The Monierave frontend is a Next.js application built from the product and
visual contracts in [`docs/phase-1`](docs/phase-1) and
[`docs/phase-2`](docs/phase-2). The implemented product includes the public
marketing experience, authentication and email-verification recovery, the
authenticated application shell, and Phase 7 account, transaction, transfer,
and beneficiary journeys.

Phase 8 adds safe profile editing, password changes, logout-all security
controls, and public support guidance. Cards, notifications, MFA, device lists,
and persisted preferences remain hidden because the backend does not provide
those contracts yet.

Phase 9 adds an isolated Three.js laboratory at `/lab/vault`. The procedural
vault-and-card scene is route-loaded, capped at a 1.5 device-pixel ratio, paused
in hidden tabs, and replaced by the existing static visual for reduced motion,
data saving, constrained hardware, or WebGL failure. It is deliberately absent
from product routes until its isolated behavior is approved.

Phase 10 promotes the adaptive visual boundary to the marketing hero and the
desktop authentication trust panel. The underlying Three.js scene remains a
separate dynamic chunk, unmounts offscreen, provides a visible pause control,
and never loads on compact viewports, reduced-motion environments, data-saving
connections, constrained devices, or unsupported WebGL contexts. Banking
screens and transactional forms remain free of 3D.

## Local setup

Requirements:

- Node.js 22 or newer
- pnpm 11.16.0
- The Go API running at `http://localhost:8080`

```bash
cp .env.example .env.local
pnpm install --frozen-lockfile
pnpm dev
```

`NEXT_PUBLIC_API_URL` is the browser-visible API origin. The API must allow the
frontend origin and credentials in its CORS configuration.

## Quality commands

```bash
pnpm format:check
pnpm lint
pnpm typecheck
pnpm test
pnpm build
pnpm build:storybook
pnpm exec playwright install chromium
pnpm test:e2e
```

The production build uses webpack explicitly. Turbopack can be evaluated later,
but pinning one verified compiler keeps local and CI behavior deterministic.

## Architecture boundaries

- `src/app` owns routing, layouts, global providers, and global styles.
- `src/components/ui` owns accessible visual primitives.
- `src/components/layout` owns the marketing, authentication, and banking
  shells and responsive application navigation.
- `src/features/auth` owns in-memory access credentials, secure session
  restoration, verified-user route boundaries, and registration flows.
- `src/features/dashboard` owns the account overview, per-account recent
  activity, and financial display formatting.
- `src/features/banking` owns account lifecycle, transaction history,
  recipient resolution, idempotent transfer intent, and beneficiary lifecycle.
- `src/features/profile` owns safe personal-information updates and supported
  session-security controls.
- `src/features/three-vault` owns the isolated capability policy, procedural 3D
  scene, failure boundary, and static fallback selection.
- `src/lib/api` owns the HTTP boundary, stable error normalization,
  cancellation, and idempotency-key support.
- `src/lib/query` owns server-state defaults and query-key factories.
- `src/test` owns MSW fixtures shared by component and integration tests.

TanStack Query is the source of truth for server state. Form state belongs in
React Hook Form, URL state belongs in the router, and ephemeral UI state stays
local until sharing it is necessary. Access tokens and sensitive session data
must never be persisted in `localStorage`; refresh credentials remain in the
backend-managed HTTP-only cookie.

API contracts are centralized manually because the backend does not yet publish
an OpenAPI schema. When it does, generated types should replace the handwritten
transport contracts.

## Design foundation

The implemented Quiet Ledger theme uses self-hosted Fraunces and Source Sans 3,
semantic light-theme tokens, tabular financial numerals, visible keyboard
focus, forced-color support, and reduced-motion fallbacks. Storybook is the
review surface for primitives before they are composed into product screens.
