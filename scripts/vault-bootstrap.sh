#!/bin/sh
set -eu

. /scripts/vault-lib.sh

ROOT_TOKEN_FILE="$VAULT_INIT_DIR/root-token"

if [ ! -s "$ROOT_TOKEN_FILE" ]; then
    echo "Vault root token file is missing: $ROOT_TOKEN_FILE" >&2
    exit 1
fi

export VAULT_TOKEN="$(cat "$ROOT_TOKEN_FILE")"

wait_for_vault

enable_secret_engine kv kv-v2
enable_secret_engine transit transit
enable_auth_method approle approle

vault kv put kv/identity-service/runtime \
    postgresql_username="$IDENTITY_POSTGRES_USER" \
    postgresql_password="$IDENTITY_POSTGRES_PASSWORD" \
    redis_password="$IDENTITY_REDIS_PASSWORD" >/dev/null
vault kv put kv/identity-service/oauth \
    google_client_secret="$IDENTITY_OAUTH_GOOGLE_CLIENT_SECRET" \
    github_client_secret="$IDENTITY_OAUTH_GITHUB_CLIENT_SECRET" >/dev/null

vault kv put kv/template-service/runtime \
    postgresql_username="$TEMPLATE_POSTGRES_USER" \
    postgresql_password="$TEMPLATE_POSTGRES_PASSWORD" \
    redis_password="$TEMPLATE_REDIS_PASSWORD" >/dev/null

vault kv put kv/organization-service/runtime \
    postgresql_username="$ORGANIZATION_POSTGRES_USER" \
    postgresql_password="$ORGANIZATION_POSTGRES_PASSWORD" \
    redis_password="$ORGANIZATION_REDIS_PASSWORD" >/dev/null

vault kv put kv/hr-service/runtime \
    postgresql_username="$HR_POSTGRES_USER" \
    postgresql_password="$HR_POSTGRES_PASSWORD" \
    redis_password="$HR_REDIS_PASSWORD" >/dev/null

vault kv put kv/access-control-service/runtime \
    postgresql_username="$ACCESS_CONTROL_POSTGRES_USER" \
    postgresql_password="$ACCESS_CONTROL_POSTGRES_PASSWORD" \
    redis_password="$ACCESS_CONTROL_REDIS_PASSWORD" >/dev/null

vault kv put kv/notification-service/runtime \
    postgresql_username="$NOTIFICATION_POSTGRES_USER" \
    postgresql_password="$NOTIFICATION_POSTGRES_PASSWORD" \
    redis_password="$NOTIFICATION_REDIS_PASSWORD" \
    aws_access_key_id="$NOTIFICATION_AWS_ACCESS_KEY_ID" \
    aws_secret_access_key="$NOTIFICATION_AWS_SECRET_ACCESS_KEY" >/dev/null

create_signing_key transit identity-access-token ecdsa-p256
create_signing_key transit identity-refresh-token ecdsa-p256
create_signing_key transit identity-id-token ecdsa-p256
create_key transit identity-totp-secret aes256-gcm96
create_key transit identity-pairwise-sub hmac key_size=32
create_key transit identity-audit-mask-hmac hmac key_size=32
create_signing_key transit identity-audit-log-sig ed25519
create_key transit template-audit-mask-hmac hmac key_size=32
create_signing_key transit template-audit-log-sig ed25519
create_key transit organization-audit-mask-hmac hmac key_size=32
create_signing_key transit organization-audit-log-sig ed25519
create_key transit hr-audit-mask-hmac hmac key_size=32
create_signing_key transit hr-audit-log-sig ed25519
create_key transit access-control-audit-mask-hmac hmac key_size=32
create_signing_key transit access-control-audit-log-sig ed25519
create_key transit notification-audit-mask-hmac hmac key_size=32
create_signing_key transit notification-audit-log-sig ed25519

for policy in \
    identity-service \
    identity-service-transit \
    template-service \
    template-service-transit \
    organization-service \
    organization-service-transit \
    hr-service \
    hr-service-transit \
    access-control-service \
    access-control-service-transit \
    notification-service \
    notification-service-transit
do
    write_policy "$policy"
done

configure_approle identity-service identity-service,identity-service-transit
configure_approle template-service template-service,template-service-transit
configure_approle organization-service organization-service,organization-service-transit
configure_approle hr-service hr-service,hr-service-transit
configure_approle access-control-service access-control-service,access-control-service-transit
configure_approle notification-service notification-service,notification-service-transit

echo "Vault KV, AppRole, and Transit are ready at $VAULT_ADDR."
