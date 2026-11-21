#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
WORK_DIR=$(mktemp -d)
trap 'rm -rf "$WORK_DIR"' EXIT

cd "$ROOT_DIR"
make env-check ENV_FILE=.env.example

tar --exclude=.git --exclude=.terraform --exclude=.env --exclude=.state \
    -cf - docker-compose.yml services messaging vault audit-archive observability .env.example \
    | tar -xf - -C "$WORK_DIR"
cp "$WORK_DIR/.env.example" "$WORK_DIR/.env"

docker compose --project-directory "$WORK_DIR" --env-file "$WORK_DIR/.env" \
    -f "$WORK_DIR/docker-compose.yml" --profile '*' config --quiet
