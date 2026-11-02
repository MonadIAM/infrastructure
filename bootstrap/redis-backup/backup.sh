#!/usr/bin/env bash
set -euo pipefail
umask 077

statefulsets=$(kubectl -n monadiam-data get statefulsets -l app.kubernetes.io/name=redis,app.kubernetes.io/part-of=monadiam -o jsonpath='{.items[*].metadata.name}')
test -n "$statefulsets"
for statefulset in $statefulsets; do
    directory="${BACKUP_DIRECTORY:?}/${statefulset%-node}"
    mkdir -p "$directory"
    snapshot="$directory/$(date -u +%F).rdb"
    trap 'rm -f "${snapshot:?}.partial"' EXIT
    printf 'Backing up %s\n' "$statefulset"
    kubectl -n monadiam-data exec -i "statefulset/$statefulset" -c redis -- bash -se -- "${statefulset%-node}" > "$snapshot.partial" <<'REDIS'
set -euo pipefail
master=$1
REDISCLI_AUTH=$(cat "${REDIS_PASSWORD_FILE:?}")
export REDISCLI_AUTH
options=(--tls --cacert "${REDIS_TLS_CA_FILE:?}" --cert "${REDIS_TLS_CERT_FILE:?}" --key "${REDIS_TLS_KEY_FILE:?}")
primary=$(redis-cli -e "${options[@]}" -p 26379 SENTINEL get-master-addr-by-name "$master")
primary=${primary%%$'\n'*}
snapshot=$(mktemp)
trap 'rm -f "${snapshot:?}"' EXIT
redis-cli -e "${options[@]}" -h "$primary" --rdb "$snapshot" >&2
redis-check-rdb "$snapshot" >&2
cat "$snapshot"
REDIS
    sync -f "$snapshot.partial"
    mv "$snapshot.partial" "$snapshot"
    find "$directory" -maxdepth 1 -type f -name '*.rdb' -printf '%f\n' | sort -r | tail -n +8 |
        while read -r expired; do rm -- "$directory/$expired"; done
done
