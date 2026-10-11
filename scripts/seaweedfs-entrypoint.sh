#!/bin/sh
set -eu

for bucket in "${SEAWEEDFS_AUDIT_LOG_BUCKET}" "${SEAWEEDFS_CHANGE_LOG_BUCKET}"; do
    case "$bucket" in
        ''|*[!a-z0-9.-]*) echo "Invalid archive bucket name" >&2; exit 1 ;;
    esac
done

sed -e "s|AUDIT_LOG_BUCKET|${SEAWEEDFS_AUDIT_LOG_BUCKET}|g" \
    -e "s|CHANGE_LOG_BUCKET|${SEAWEEDFS_CHANGE_LOG_BUCKET}|g" \
    /config/credentials.json > /tmp/credentials.json

exec /entrypoint.sh "$@"
