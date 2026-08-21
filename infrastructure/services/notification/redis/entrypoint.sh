#!/bin/sh
set -eu

TLS_DIR=/tmp/redis-tls

mkdir -p "$TLS_DIR"
cp /tls/ca.crt "$TLS_DIR/ca.crt"
cp /tls/redis.crt "$TLS_DIR/redis.crt"
cp /tls/redis.key "$TLS_DIR/redis.key"
cp /tls/client.crt "$TLS_DIR/client.crt"
cp /tls/client.key "$TLS_DIR/client.key"
chown -R redis:redis "$TLS_DIR"
chmod 0444 "$TLS_DIR/ca.crt" "$TLS_DIR/redis.crt" "$TLS_DIR/client.crt"
chmod 0400 "$TLS_DIR/redis.key" "$TLS_DIR/client.key"

exec docker-entrypoint.sh redis-server \
    --port 0 \
    --tls-port "${REDIS_PORT:?}" \
    --tls-cert-file "$TLS_DIR/redis.crt" \
    --tls-key-file "$TLS_DIR/redis.key" \
    --tls-ca-cert-file "$TLS_DIR/ca.crt" \
    --tls-auth-clients yes \
    --requirepass "$(cat /vault/secrets/redis_password)"
