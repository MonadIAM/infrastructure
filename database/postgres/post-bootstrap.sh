#!/bin/bash
set -eu

readonly_role="$(printf '%s' "${POSTGRES_DB}" | tr '-' '_')_readonly"

psql "$1" \
    -v ON_ERROR_STOP=1 \
    -v postgres_db="${POSTGRES_DB}" \
    -v postgres_user="${POSTGRES_USER}" \
    -v readonly_role="${readonly_role}" \
    -f /docker-entrypoint-initdb.d/post-bootstrap.sql
