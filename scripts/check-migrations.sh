#!/usr/bin/env bash

set -euo pipefail

database_url="${DB_TEST_SOURCE:-}"
if [[ -z "${database_url}" ]]; then
  echo "DB_TEST_SOURCE is required" >&2
  exit 1
fi

database_without_query="${database_url%%\?*}"
database_name="${database_without_query##*/}"
if [[ "${database_name}" != *_test ]]; then
  echo "refusing migration cycle against '${database_name}'; use a dedicated *_test database" >&2
  exit 1
fi

migrate -path db/migration -database "${database_url}" -verbose up
migrate -path db/migration -database "${database_url}" -verbose down -all
migrate -path db/migration -database "${database_url}" -verbose up
