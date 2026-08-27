#!/bin/sh
set -eu

SCRIPT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
ROOT_DIR="$(CDPATH= cd -- "$SCRIPT_DIR/../.." && pwd)"

. "$ROOT_DIR/scripts/lib/env.sh"
. "$ROOT_DIR/scripts/lib/tools.sh"
. "$ROOT_DIR/scripts/lib/cluster.sh"
. "$ROOT_DIR/scripts/lib/stack.sh"

load_env
require_env CLUSTER
check_required_tools

if ! cluster_exists; then
    echo "k3d cluster '$CLUSTER' does not exist. Use: make start" >&2
    exit 1
fi

start_or_create_cluster
reconcile_platform_stack
show_cluster_status
