#!/bin/sh
set -eu

: "${CONSUL_CA_INTERMEDIATE_CERT_TTL:?}"
: "${CONSUL_CA_PRIVATE_KEY_BITS:?}"
: "${CONSUL_CA_LEAF_CERT_TTL:?}"
: "${CONSUL_HTTP_ADDR:?}"
: "${VAULT_TOKEN_FILE:?}"
: "${VAULT_CACERT:?}"
: "${VAULT_ADDR:?}"

export CONSUL_HTTP_ADDR

i=0
until consul info >/dev/null 2>&1; do
    i=$((i + 1))
    if [ "$i" -ge 60 ]; then
        echo "Timed out waiting for Consul" >&2
        exit 1
    fi
    sleep 1
done

i=0
until [ -s "$VAULT_TOKEN_FILE" ]; do
    i=$((i + 1))
    if [ "$i" -ge 60 ]; then
        echo "Timed out waiting for $VAULT_TOKEN_FILE" >&2
        exit 1
    fi
    sleep 1
done

i=0
until [ -s "$VAULT_CACERT" ]; do
    i=$((i + 1))
    if [ "$i" -ge 60 ]; then
        echo "Timed out waiting for $VAULT_CACERT" >&2
        exit 1
    fi
    sleep 1
done

VAULT_TOKEN="$(cat "$VAULT_TOKEN_FILE")"

cat > /tmp/consul-connect-ca.json <<EOF
{
  "Provider": "vault",
  "Config": {
    "Address": "${VAULT_ADDR}",
    "Token": "${VAULT_TOKEN}",
    "RootPKIPath": "pki-root",
    "IntermediatePKIPath": "pki",
    "LeafCertTTL": "${CONSUL_CA_LEAF_CERT_TTL}",
    "IntermediateCertTTL": "${CONSUL_CA_INTERMEDIATE_CERT_TTL}",
    "PrivateKeyType": "rsa",
    "PrivateKeyBits": ${CONSUL_CA_PRIVATE_KEY_BITS},
    "CAFile": "${VAULT_CACERT}",
    "TLSServerName": "vault"
  }
}
EOF

consul connect ca set-config -config-file /tmp/consul-connect-ca.json
