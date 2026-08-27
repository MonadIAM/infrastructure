#!/bin/sh
set -eu

SCRIPT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
ROOT_DIR="$(CDPATH= cd -- "$SCRIPT_DIR/../.." && pwd)"

. "$ROOT_DIR/scripts/lib/env.sh"
. "$ROOT_DIR/scripts/lib/vault.sh"

load_env
require_env KUBE_CONTEXT VAULT_NAMESPACE VAULT_POD VAULT_ADDR VAULT_CACERT

vault_exec vault status
