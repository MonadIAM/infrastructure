#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
WORK_DIR=$(mktemp -d)
trap 'rm -rf "$WORK_DIR"' EXIT

cd "$ROOT_DIR"
terraform fmt -check -recursive terraform
export TF_IN_AUTOMATION=1

if [ -n "${TF_PLUGIN_CACHE_DIR:-}" ]; then
    mkdir -p "$TF_PLUGIN_CACHE_DIR"
fi

for root in terraform/*; do
    [ -f "$root/versions.tf" ] || continue
    export TF_DATA_DIR="$WORK_DIR/${root##*/}"
    terraform -chdir="$root" init -backend=false -input=false -lockfile=readonly -no-color
    terraform -chdir="$root" validate -no-color
done
