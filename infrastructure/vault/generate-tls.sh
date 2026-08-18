#!/bin/sh
set -eu

CERT_DIR=/vault/tls
CA_DIR=/vault/ca
DAYS=3650

mkdir -p "$CERT_DIR"
mkdir -p "$CA_DIR"

if [ -f "$CERT_DIR/vault.crt" ] && [ -f "$CERT_DIR/vault.key" ] && [ -f "$CERT_DIR/ca.crt" ]; then
    cp "$CERT_DIR/ca.crt" "$CA_DIR/ca.crt"
    chmod 0444 "$CA_DIR/ca.crt"
    echo "TLS material already present in $CERT_DIR, skipping generation."
    exit 0
fi

echo "Generating local CA and Vault server certificate (dev/stand-only, self-signed - see infra/VAULT-DEV-MODE-TRANSITION-PLAN.md)..."

openssl genrsa -out "$CERT_DIR/ca.key" 4096
openssl req -x509 -new -nodes -key "$CERT_DIR/ca.key" -sha256 -days "$DAYS" \
    -subj "/CN=MonadIAM Local Vault CA" -out "$CERT_DIR/ca.crt"

openssl genrsa -out "$CERT_DIR/vault.key" 2048

cat > "$CERT_DIR/vault.ext" <<EOF
subjectAltName = DNS:vault, DNS:localhost, IP:127.0.0.1
extendedKeyUsage = serverAuth
EOF

openssl req -new -key "$CERT_DIR/vault.key" -subj "/CN=vault" \
    | openssl x509 -req -CA "$CERT_DIR/ca.crt" -CAkey "$CERT_DIR/ca.key" -CAcreateserial \
        -days "$DAYS" -sha256 -extfile "$CERT_DIR/vault.ext" -out "$CERT_DIR/vault.crt"

rm -f "$CERT_DIR/vault.ext" "$CERT_DIR/ca.srl"

chmod 0400 "$CERT_DIR/ca.key" "$CERT_DIR/vault.key"
chmod 0444 "$CERT_DIR/ca.crt" "$CERT_DIR/vault.crt"
cp "$CERT_DIR/ca.crt" "$CA_DIR/ca.crt"
chmod 0444 "$CA_DIR/ca.crt"

echo "TLS material ready in $CERT_DIR."
