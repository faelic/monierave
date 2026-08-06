#!/usr/bin/env bash

set -euo pipefail

if ! command -v sqlc >/dev/null 2>&1; then
  echo "sqlc is required" >&2
  exit 1
fi

before="$(git status --porcelain --untracked-files=all -- db/sqlc)"
sqlc generate
after="$(git status --porcelain --untracked-files=all -- db/sqlc)"

if [[ "${before}" != "${after}" ]]; then
  echo "sqlc generated files are out of date; run 'sqlc generate' and commit the result" >&2
  git status --short -- db/sqlc >&2
  git diff -- db/sqlc >&2
  exit 1
fi
