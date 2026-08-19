#!/bin/sh
set -eu

exec docker-entrypoint.sh redis-server --port "${REDIS_PORT:-6379}" --requirepass "$(cat /vault/secrets/redis_password)"
