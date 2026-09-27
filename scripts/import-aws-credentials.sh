#!/bin/sh
set -eu

SCRIPT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
ROOT_DIR="$(CDPATH= cd -- "$SCRIPT_DIR/.." && pwd)"

if [ ! -f "$ROOT_DIR/.env" ]; then
    echo ".env not found: $ROOT_DIR/.env" >&2
    exit 1
fi

credentials="$(sed -n '2p' "$ROOT_DIR/aws.csv" | tr -d '\r"')"
access_key_id="${credentials%%,*}"
secret_access_key="${credentials#*,}"
temporary="$(mktemp "$ROOT_DIR/.env.XXXXXX")"
trap 'rm -f "$temporary"' EXIT

sed \
    -e "s|^NOTIFICATION_AWS_ACCESS_KEY_ID=.*|NOTIFICATION_AWS_ACCESS_KEY_ID=\"$access_key_id\"|" \
    -e "s|^NOTIFICATION_AWS_SECRET_ACCESS_KEY=.*|NOTIFICATION_AWS_SECRET_ACCESS_KEY=\"$secret_access_key\"|" \
    "$ROOT_DIR/.env" > "$temporary"

mv "$temporary" "$ROOT_DIR/.env"
