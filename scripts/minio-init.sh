#!/bin/sh
set -eu

echo "Waiting for MinIO at http://minio:${MINIO_API_INTERNAL_PORT}..."
i=0
until mc alias set local http://minio:${MINIO_API_INTERNAL_PORT} "$MINIO_ROOT_USER" "$MINIO_ROOT_PASSWORD" >/dev/null 2>&1; do
    i=$((i + 1))
    if [ "$i" -ge 60 ]; then
        echo "MinIO did not become ready in time (http://minio:${MINIO_API_INTERNAL_PORT})" >&2
        exit 1
    fi
    sleep 1
done

mc ls "local/$MINIO_AUDIT_LOG_BUCKET" >/dev/null 2>&1 || mc mb --with-lock "local/$MINIO_AUDIT_LOG_BUCKET"
mc ls "local/$MINIO_CHANGE_LOG_BUCKET" >/dev/null 2>&1 || mc mb --with-lock "local/$MINIO_CHANGE_LOG_BUCKET"

mc retention set --default COMPLIANCE "${MINIO_AUDIT_LOG_RETENTION_DAYS}d" "local/$MINIO_AUDIT_LOG_BUCKET"
mc retention set --default COMPLIANCE "${MINIO_CHANGE_LOG_RETENTION_DAYS}d" "local/$MINIO_CHANGE_LOG_BUCKET"

mc ilm rule add --expire-days "$MINIO_AUDIT_LOG_RETENTION_DAYS" "local/$MINIO_AUDIT_LOG_BUCKET" 2>/dev/null || true
mc ilm rule add --expire-days "$MINIO_CHANGE_LOG_RETENTION_DAYS" "local/$MINIO_CHANGE_LOG_BUCKET" 2>/dev/null || true

mc admin policy create local archiver-policy /minio/policy.archiver.json 2>/dev/null || true
mc admin user add local "$MINIO_ARCHIVER_ACCESS_KEY" "$MINIO_ARCHIVER_SECRET_KEY" 2>/dev/null || true
mc admin policy attach local archiver-policy --user "$MINIO_ARCHIVER_ACCESS_KEY"

echo "MinIO buckets/policy/user ready."
