#!/bin/sh
set -eu

cd /workspace

terraform init -input=false
terraform apply -auto-approve -input=false

mkdir -p .output
for service in identity-service access-control-service notification-service; do
    key="$(printf '%s' "$service" | tr '-' '_')"
    terraform output -raw "${key}_provisioner_token" > ".output/provisioner-token-${service}"
done
