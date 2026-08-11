# Monierave Deployment Checklist

Use this checklist before every production-like deployment. Do not deploy while
any blocking item remains incomplete.

## Security Configuration

- [ ] Set `APP_ENV=production`.
- [ ] Generate a cryptographically random `OPERATIONS_TOKEN` containing at least
  32 characters. Store it in the deployment platform's secret manager, never in
  Git or a frontend environment variable.
- [ ] Configure every allowed frontend origin as an explicit HTTPS origin. Do
  not use wildcards.
- [ ] Configure `DB_SOURCE` with PostgreSQL TLS using `sslmode=require`,
  `verify-ca`, or preferably `verify-full` with the provider's CA certificate.
- [ ] Configure an HTTPS `PUBLIC_BASE_URL` and a production Resend webhook
  signing secret containing at least 32 characters.
- [ ] Confirm secure refresh cookies and email verification are enabled.

Generate an operations token locally with:

```bash
openssl rand -base64 48
```

## Database

- [ ] Back up the target database and confirm the restore procedure.
- [ ] Apply migration `000023_security_hardening` before starting the new API or
  worker image.
- [ ] Confirm `email_jobs.verification_token_hash` and
  `verification_token_expires_at` exist after migration.
- [ ] Confirm the application database user has only the permissions needed by
  the API, worker, and migration process.

## Release Verification

- [ ] Run `make generated-check`.
- [ ] Run `go vet ./...`.
- [ ] Run `go test -race -count=1 ./...` against the dedicated test database.
- [ ] Run `govulncheck ./...`.
- [ ] Run the frontend formatter, lint, typecheck, tests, and production build.
- [ ] Build the production Docker image.
- [ ] Verify `/readyz` and `/metrics` return `404` without the operations token
  and succeed with the correct bearer token.
- [ ] Complete a signup, email verification, login, refresh rotation, logout,
  and password/email-change smoke test in the deployed environment.
- [ ] Confirm logs contain no access tokens, refresh tokens, verification
  tokens, session IDs, email addresses, raw IP addresses, or raw user agents.

## Deployment Reminder

When deployment work begins, ask Codex to review this checklist against the
target environment before applying migrations or releasing containers.

