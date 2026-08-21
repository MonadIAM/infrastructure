#!/bin/sh
set -eu

: "${CERT_DIR:?}"
: "${CA_DIR:?}"
: "${DAYS:?}"

CA_KEY="$CERT_DIR/ca.key"
CA_CERT="$CERT_DIR/ca.crt"
VAULT_KEY="$CERT_DIR/vault.key"
VAULT_CERT="$CERT_DIR/vault.crt"
VAULT_EXT="$CERT_DIR/vault.ext"

mkdir -p "$CERT_DIR" "$CA_DIR"

if [ -s "$CA_KEY" ] && [ -s "$CA_CERT" ] && [ -s "$VAULT_KEY" ] && [ -s "$VAULT_CERT" ]; then
    echo "Vault HTTPS TLS material already exists in $CERT_DIR."
else
    echo "Generating Vault HTTPS TLS material in $CERT_DIR."

    umask 077
    tmp_dir="$CERT_DIR/.generate.$$"
    rm -rf "$tmp_dir"
    mkdir -p "$tmp_dir"
    trap 'rm -rf "$tmp_dir"' EXIT HUP INT TERM

    openssl genrsa -out "$tmp_dir/ca.key" 4096
    openssl req -x509 -new -nodes -key "$tmp_dir/ca.key" -sha256 -days "$DAYS" \
        -subj "/CN=MonadIAM Local Vault CA" \
        -out "$tmp_dir/ca.crt"

    openssl genrsa -out "$tmp_dir/vault.key" 2048

    cat > "$tmp_dir/vault.ext" <<'EOF'
subjectAltName = DNS:vault,DNS:localhost,IP:127.0.0.1
extendedKeyUsage = serverAuth
keyUsage = digitalSignature,keyEncipherment
EOF

    openssl req -new -key "$tmp_dir/vault.key" -subj "/CN=vault" \
        | openssl x509 -req \
            -CA "$tmp_dir/ca.crt" \
            -CAkey "$tmp_dir/ca.key" \
            -CAcreateserial \
            -days "$DAYS" \
            -sha256 \
            -extfile "$tmp_dir/vault.ext" \
            -out "$tmp_dir/vault.crt"

    cp "$tmp_dir/ca.key" "$CA_KEY"
    cp "$tmp_dir/ca.crt" "$CA_CERT"
    cp "$tmp_dir/vault.key" "$VAULT_KEY"
    cp "$tmp_dir/vault.crt" "$VAULT_CERT"
    rm -f "$VAULT_EXT"

    rm -rf "$tmp_dir"
    trap - EXIT HUP INT TERM
fi

chmod 0400 "$CA_KEY" "$VAULT_KEY"
chmod 0444 "$CA_CERT" "$VAULT_CERT"

cp "$CA_CERT" "$CA_DIR/ca.crt"
chmod 0444 "$CA_DIR/ca.crt"

echo "Vault HTTPS CA is available at $CA_DIR/ca.crt."
