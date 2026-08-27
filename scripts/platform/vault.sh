#!/bin/sh
set -eu

SCRIPT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
ROOT_DIR="$(CDPATH= cd -- "$SCRIPT_DIR/../.." && pwd)"

. "$ROOT_DIR/scripts/lib/env.sh"
. "$ROOT_DIR/scripts/lib/vault.sh"

load_env
require_vault_env

cd "$ROOT_DIR"

umask 077
mkdir -p "$VAULT_STATE_DIR"
chmod 700 "$VAULT_STATE_DIR"

kubectl --context "$KUBE_CONTEXT" apply -k clusters/local

helm repo add hashicorp "$HASHICORP_HELM_REPO" --force-update
helm repo update hashicorp

kubectl --context "$KUBE_CONTEXT" apply -k platform/vault/resources
kubectl --context "$KUBE_CONTEXT" wait --for=condition=Ready certificate/vault-server-tls -n "$VAULT_NAMESPACE" --timeout=180s

helm upgrade --install vault hashicorp/vault \
    --version "$VAULT_CHART_VERSION" \
    --namespace "$VAULT_NAMESPACE" \
    --create-namespace \
    --kube-context "$KUBE_CONTEXT" \
    --values platform/vault/helm/values.yaml

kubectl --context "$KUBE_CONTEXT" wait --for=create "pod/$VAULT_POD" -n "$VAULT_NAMESPACE" --timeout=180s
kubectl --context "$KUBE_CONTEXT" wait --for=condition=PodScheduled "pod/$VAULT_POD" -n "$VAULT_NAMESPACE" --timeout=180s
kubectl --context "$KUBE_CONTEXT" wait --for=condition=Ready "pod/$VAULT_POD" -n "$VAULT_NAMESPACE" --timeout=300s

initialize_vault
unseal_vault
bootstrap_vault

helm upgrade --install vault-secrets-operator hashicorp/vault-secrets-operator \
    --version "$VAULT_SECRETS_OPERATOR_VERSION" \
    --namespace vault-secrets-operator \
    --create-namespace \
    --kube-context "$KUBE_CONTEXT" \
    --values infrastructure/security/vault-secrets-operator/helm/values.yaml

kubectl --context "$KUBE_CONTEXT" rollout status deployment/vault-secrets-operator-controller-manager -n vault-secrets-operator --timeout=180s
kubectl --context "$KUBE_CONTEXT" apply -k infrastructure/security/vault-secrets-operator/resources
kubectl --context "$KUBE_CONTEXT" rollout restart deployment/vault-secrets-operator-controller-manager -n vault-secrets-operator
kubectl --context "$KUBE_CONTEXT" rollout status deployment/vault-secrets-operator-controller-manager -n vault-secrets-operator --timeout=180s
kubectl --context "$KUBE_CONTEXT" wait --for=condition=Ready certificate/vault-server-ca -n vault-secrets-operator --timeout=180s
kubectl --context "$KUBE_CONTEXT" wait --for=condition=Ready vaultconnections.secrets.hashicorp.com/vault -n vault-secrets-operator --timeout=180s
kubectl --context "$KUBE_CONTEXT" wait --for=condition=Ready vaultauths.secrets.hashicorp.com/monadiam-apps -n monadiam-apps --timeout=180s
kubectl --context "$KUBE_CONTEXT" wait --for=condition=Ready vaultauths.secrets.hashicorp.com/monadiam-data -n monadiam-data --timeout=180s
kubectl --context "$KUBE_CONTEXT" wait --for=condition=Ready vaultauths.secrets.hashicorp.com/monadiam-messaging -n monadiam-messaging --timeout=180s
kubectl --context "$KUBE_CONTEXT" wait --for=condition=Ready vaultauths.secrets.hashicorp.com/monadiam-observability -n monadiam-observability --timeout=180s
kubectl --context "$KUBE_CONTEXT" wait --for=condition=Ready vaultauths.secrets.hashicorp.com/clickhouse-operator -n clickhouse-operator --timeout=180s

wait_vault_static_secret_reconciled monadiam-apps monadiam-bootstrap monadiam-bootstrap
wait_vault_static_secret_reconciled monadiam-data monadiam-data-bootstrap monadiam-data-bootstrap
wait_vault_static_secret_reconciled monadiam-messaging monadiam-messaging-bootstrap monadiam-messaging-bootstrap
wait_vault_static_secret_reconciled monadiam-observability grafana-admin grafana-admin
wait_vault_static_secret_reconciled monadiam-data postgresql-main-superuser postgresql-main-superuser
wait_vault_static_secret_reconciled monadiam-data postgresql-main-app postgresql-main-app
wait_vault_static_secret_reconciled monadiam-data postgresql-main-debezium postgresql-main-debezium
wait_vault_static_secret_reconciled monadiam-messaging kafka-main-admin-password kafka-main-admin-password
wait_vault_static_secret_reconciled monadiam-data redis-main-auth redis-main-auth
wait_vault_static_secret_reconciled monadiam-data clickhouse-main-app clickhouse-main-app
wait_vault_static_secret_reconciled monadiam-data clickhouse-main-intercluster clickhouse-main-intercluster
wait_vault_static_secret_reconciled clickhouse-operator clickhouse-operator-credentials clickhouse-operator-altinity-clickhouse-operator
wait_vault_static_secret_reconciled monadiam-data minio-main-root minio-main-root
wait_vault_static_secret_reconciled monadiam-data minio-main-archiver minio-main-archiver

kubectl --context "$KUBE_CONTEXT" get pods -n "$VAULT_NAMESPACE"
kubectl --context "$KUBE_CONTEXT" get vaultconnections.secrets.hashicorp.com -n vault-secrets-operator
kubectl --context "$KUBE_CONTEXT" get vaultauths.secrets.hashicorp.com -A
kubectl --context "$KUBE_CONTEXT" get secret monadiam-bootstrap -n monadiam-apps
kubectl --context "$KUBE_CONTEXT" get secret monadiam-data-bootstrap -n monadiam-data
kubectl --context "$KUBE_CONTEXT" get secret monadiam-messaging-bootstrap -n monadiam-messaging
kubectl --context "$KUBE_CONTEXT" get secret grafana-admin -n monadiam-observability
kubectl --context "$KUBE_CONTEXT" get secret postgresql-main-superuser postgresql-main-app postgresql-main-debezium redis-main-auth clickhouse-main-app clickhouse-main-intercluster -n monadiam-data
kubectl --context "$KUBE_CONTEXT" get secret minio-main-root minio-main-archiver -n monadiam-data
kubectl --context "$KUBE_CONTEXT" get secret kafka-main-admin-password -n monadiam-messaging
kubectl --context "$KUBE_CONTEXT" get secret clickhouse-operator-altinity-clickhouse-operator -n clickhouse-operator
