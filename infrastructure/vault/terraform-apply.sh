#!/bin/sh
set -eu

: "${VAULT_ADDR:?}"
: "${VAULT_CACERT:?}"
: "${VAULT_ROOT_TOKEN_FILE:?}"
: "${TERRAFORM_OUTPUT_DIR:?}"

cd /workspace

wait_for_file() {
    path="$1"
    i=0
    until [ -s "$path" ]; do
        i=$((i + 1))
        if [ "$i" -ge 60 ]; then
            echo "Timed out waiting for $path" >&2
            exit 1
        fi
        sleep 1
    done
}

wait_for_file "$VAULT_ROOT_TOKEN_FILE"

TF_VAR_vault_bootstrap_token="$(cat "$VAULT_ROOT_TOKEN_FILE")"
export TF_VAR_vault_bootstrap_token


terraform init -input=false
terraform apply -auto-approve -input=false

output_tmp="$TERRAFORM_OUTPUT_DIR.tmp"
rm -rf "$output_tmp"
mkdir -p "$output_tmp"

for service in identity-service access-control-service notification-service; do
    key="$(printf '%s' "$service" | tr '-' '_')"
    terraform output -raw "${key}_provisioner_token" > "$output_tmp/provisioner-token-$service"
done

terraform output -raw consul_connect_ca_token > "$output_tmp/consul-connect-ca-token"
terraform output -raw service_tls_issuer_token > "$output_tmp/service-tls-issuer-token"
terraform output -raw service_tls_ca_bundle > "$output_tmp/service-tls-ca.crt"
terraform output -raw infra_agent_bootstrap_token > "$output_tmp/infra-agent-bootstrap-token"

chmod 0400 "$output_tmp"/*
rm -rf "$TERRAFORM_OUTPUT_DIR"
mv "$output_tmp" "$TERRAFORM_OUTPUT_DIR"
