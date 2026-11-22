#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
WORK_DIR=$(mktemp -d)
trap 'rm -rf "$WORK_DIR"' EXIT
CACHE_DIR="${XDG_CACHE_HOME:-$HOME/.cache}/monadiam/terraform-check"

cd "$ROOT_DIR"
terraform fmt -check -recursive terraform
export TF_IN_AUTOMATION=1
export TF_PLUGIN_CACHE_DIR="${TF_PLUGIN_CACHE_DIR:-$CACHE_DIR/providers}"
mkdir -p "$TF_PLUGIN_CACHE_DIR"

for root in terraform/*; do
    [ -f "$root/versions.tf" ] || continue
    export TF_DATA_DIR="$CACHE_DIR/data/${root##*/}"
    if [ -d "$TF_DATA_DIR/providers" ] && terraform -chdir="$root" validate -no-color >"$WORK_DIR/validate" 2>&1; then
        cat "$WORK_DIR/validate"
        continue
    fi
    terraform -chdir="$root" init -backend=false -input=false -lockfile=readonly -no-color
    terraform -chdir="$root" validate -no-color
done
