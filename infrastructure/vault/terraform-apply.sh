#!/bin/sh
set -eu

cd /workspace

TF_VAR_vault_bootstrap_token="$(cat /vault/init-keys/root-token)"
export TF_VAR_vault_bootstrap_token

terraform init -input=false
terraform apply -auto-approve -input=false

mkdir -p .output
for service in identity-service access-control-service notification-service; do
    key="$(printf '%s' "$service" | tr '-' '_')"
    terraform output -raw "${key}_provisioner_token" > ".output/provisioner-token-${service}"
done

terraform output -raw consul_connect_ca_token > .output/consul-connect-ca-token
