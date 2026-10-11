#!/bin/sh
set -eu

export MC_CONFIG_DIR=/tmp/mc

if [ "$SEAWEEDFS_ADMIN_ACCESS_KEY" = "$SEAWEEDFS_ARCHIVER_ACCESS_KEY" ]; then
    echo "Admin and archiver must use different access keys" >&2
    exit 1
fi

echo "Waiting for SeaweedFS at ${SEAWEEDFS_ENDPOINT}..."
i=0
until mc alias set local "$SEAWEEDFS_ENDPOINT" "$SEAWEEDFS_ADMIN_ACCESS_KEY" "$SEAWEEDFS_ADMIN_SECRET_KEY" --api S3v4 --path on >/dev/null 2>&1 && mc ls local >/dev/null 2>&1; do
    i=$((i + 1))
    if [ "$i" -ge 60 ]; then
        echo "SeaweedFS did not become ready in time: ${SEAWEEDFS_ENDPOINT}" >&2
        exit 1
    fi
    sleep 2
done

render_template() {
    while IFS= read -r line; do
        case "$line" in
            *"$1"*) printf '%s%s%s\n' "${line%%"$1"*}" "$2" "${line#*"$1"}" ;;
            *) printf '%s\n' "$line" ;;
        esac
    done < "$3"
}

configure_bucket() {
    bucket="$1"
    days="$2"

    case "$bucket" in
        ''|*[!a-z0-9.-]*) echo "Invalid archive bucket name: $bucket" >&2; exit 1 ;;
    esac
    case "$days" in
        ''|0*|*[!0-9]*) echo "Retention must be a positive integer" >&2; exit 1 ;;
    esac

    mc mb --ignore-existing --with-lock --region "$SEAWEEDFS_REGION" "local/$bucket"
    mc version enable "local/$bucket"
    mc retention set --default COMPLIANCE "${days}d" "local/$bucket"

    render_template BUCKET_NAME "$bucket" /config/policy.archiver.json > /tmp/archiver-policy.json
    mc anonymous set-json /tmp/archiver-policy.json "local/$bucket"
    render_template '"RETENTION_DAYS"' "$days" /config/lifecycle.json > /tmp/lifecycle.json
    mc ilm rule import "local/$bucket" < /tmp/lifecycle.json

    mc retention info --default "local/$bucket"
    mc ilm rule ls "local/$bucket"
}

configure_bucket "$SEAWEEDFS_AUDIT_LOG_BUCKET" "$SEAWEEDFS_AUDIT_LOG_RETENTION_DAYS"
configure_bucket "$SEAWEEDFS_CHANGE_LOG_BUCKET" "$SEAWEEDFS_CHANGE_LOG_RETENTION_DAYS"
echo "SeaweedFS archive buckets ready."
