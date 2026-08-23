#!/bin/bash
set -e

: "${CLICKHOUSE_NATIVE_INTERNAL_PORT:?}"

wait_for_file() {
    path="$1"
    i=0
    until [ -s "$path" ]; do
        i=$((i + 1))
        if [ "$i" -ge 60 ]; then
            echo "Timed out waiting for $path" >&2
            exit 1
        fi
        sleep 1
    done
}

wait_for_file /tls/ca.crt

mkdir -p /etc/clickhouse-client
cat > /etc/clickhouse-client/config.xml <<EOF
<clickhouse>
    <openSSL>
        <client>
            <loadDefaultCAFile>false</loadDefaultCAFile>
            <caConfig>/tls/ca.crt</caConfig>
            <verificationMode>strict</verificationMode>
            <invalidCertificateHandler>
                <name>RejectCertificateHandler</name>
            </invalidCertificateHandler>
        </client>
    </openSSL>
</clickhouse>
EOF

CLICKHOUSE_PASSWORD="$(cat /vault/secrets/clickhouse_password)"
CLICKHOUSE_WRITER_PASSWORD="$(cat /vault/secrets/clickhouse_writer_password)"
CLICKHOUSE_READER_PASSWORD="$(cat /vault/secrets/clickhouse_reader_password)"

CLICKHOUSE="clickhouse-client --host clickhouse --port ${CLICKHOUSE_NATIVE_INTERNAL_PORT} --secure --user ${CLICKHOUSE_USER} --password ${CLICKHOUSE_PASSWORD} --multiquery"

for file in /schema/*.sql; do
    echo "Applying $file"
    sed \
        -e "s|{{writer_user}}|${CLICKHOUSE_WRITER_USER}|g" \
        -e "s|{{writer_password}}|${CLICKHOUSE_WRITER_PASSWORD}|g" \
        -e "s|{{reader_user}}|${CLICKHOUSE_READER_USER}|g" \
        -e "s|{{reader_password}}|${CLICKHOUSE_READER_PASSWORD}|g" \
        "$file" | $CLICKHOUSE
done
