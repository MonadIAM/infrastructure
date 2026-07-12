#!/bin/sh
set -eu

export CLICKHOUSE_PASSWORD="$(cat /vault/secrets/clickhouse_password)"

exec /entrypoint.sh "$@"
