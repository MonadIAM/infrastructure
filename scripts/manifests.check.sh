#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
WORK_DIR=$(mktemp -d)
trap 'rm -rf "$WORK_DIR"' EXIT

cd "$ROOT_DIR"

while IFS= read -r manifest; do
    kubectl kustomize "${manifest%/*}" > /dev/null
done < <(find applications infrastructure clusters -name kustomization.yaml -print)

for manifest in clusters/dev/applications/*/kustomization.yaml clusters/dev/infrastructure/*/kustomization.yaml clusters/dev/argocd/kustomization.yaml; do
    [ "$manifest" != clusters/dev/infrastructure/system/kustomization.yaml ] || continue
    kubectl kustomize "${manifest%/*}"
    printf '%s\n' '---'
done > "$WORK_DIR/manifests.yaml"

kubeconform -strict -summary -kubernetes-version 1.35.5 \
    -schema-location 'https://raw.githubusercontent.com/yannh/kubernetes-json-schema/102361970337d65537d40141dc29b991b8e95786/{{.NormalizedKubernetesVersion}}-standalone{{.StrictSuffix}}/{{.ResourceKind}}{{.KindSuffix}}.json' \
    -schema-location 'https://raw.githubusercontent.com/datreeio/CRDs-catalog/c86448cbf1a840ca62d119259864863897c8748a/{{.Group}}/{{.ResourceKind}}_{{.ResourceAPIVersion}}.json' \
    "$WORK_DIR/manifests.yaml"

helm lint infrastructure/gateway-api/chart --strict --kube-version 1.35.5 \
    --set-string applicationsIP=127.0.0.1
helm lint infrastructure/seaweedfs/chart --strict --kube-version 1.35.5 \
    --values clusters/dev/infrastructure/seaweedfs/bootstrap-values.yaml \
    --set-string image=monadiam/stand-cli:ci
