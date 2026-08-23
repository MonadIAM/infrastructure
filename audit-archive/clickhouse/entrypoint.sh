#!/bin/sh
set -eu

: "${CLICKHOUSE_HTTP_INTERNAL_PORT:?}"
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

wait_for_file /vault/secrets/clickhouse_password
wait_for_file /vault/secrets/kafka_sasl_password
wait_for_file /tls/ca.crt
wait_for_file /tls/client.crt
wait_for_file /tls/client.key
wait_for_file /tls/clickhouse.crt
wait_for_file /tls/clickhouse.key

export CLICKHOUSE_PASSWORD="$(cat /vault/secrets/clickhouse_password)"
KAFKA_SASL_PASSWORD="$(tr -d '\r\n' < /vault/secrets/kafka_sasl_password)"

TLS_DIR=/tmp/clickhouse-tls
mkdir -p "$TLS_DIR"
cp /tls/ca.crt /tls/client.crt /tls/client.key /tls/clickhouse.crt /tls/clickhouse.key "$TLS_DIR/"
chown clickhouse:clickhouse "$TLS_DIR" "$TLS_DIR"/*
chmod 0444 "$TLS_DIR/ca.crt" "$TLS_DIR/client.crt" "$TLS_DIR/clickhouse.crt"
chmod 0400 "$TLS_DIR/client.key" "$TLS_DIR/clickhouse.key"

mkdir -p /etc/clickhouse-server/config.d /etc/clickhouse-client

cat > /etc/clickhouse-server/config.d/tls.xml <<EOF
<clickhouse>
    <https_port>${CLICKHOUSE_HTTP_INTERNAL_PORT}</https_port>
    <tcp_port_secure>${CLICKHOUSE_NATIVE_INTERNAL_PORT}</tcp_port_secure>
    <tcp_port remove="true"/>
    <http_port remove="true"/>

    <openSSL>
        <server>
            <certificateFile>${TLS_DIR}/clickhouse.crt</certificateFile>
            <privateKeyFile>${TLS_DIR}/clickhouse.key</privateKeyFile>
            <caConfig>${TLS_DIR}/ca.crt</caConfig>
            <verificationMode>none</verificationMode>
            <loadDefaultCAFile>false</loadDefaultCAFile>
            <cacheSessions>true</cacheSessions>
            <disableProtocols>sslv2,sslv3</disableProtocols>
            <preferServerCiphers>true</preferServerCiphers>
        </server>
    </openSSL>
</clickhouse>
EOF

cat > /etc/clickhouse-client/config.xml <<EOF
<clickhouse>
    <port>${CLICKHOUSE_NATIVE_INTERNAL_PORT}</port>
    <secure>true</secure>
    <openSSL>
        <client>
            <loadDefaultCAFile>false</loadDefaultCAFile>
            <caConfig>${TLS_DIR}/ca.crt</caConfig>
            <verificationMode>strict</verificationMode>
            <invalidCertificateHandler>
                <name>RejectCertificateHandler</name>
            </invalidCertificateHandler>
        </client>
    </openSSL>
</clickhouse>
EOF

cat > /etc/clickhouse-server/config.d/kafka.xml <<EOF
<clickhouse>
    <kafka>
        <security_protocol>sasl_ssl</security_protocol>
        <sasl_mechanism>SCRAM-SHA-512</sasl_mechanism>
        <sasl_username>clickhouse</sasl_username>
        <sasl_password>${KAFKA_SASL_PASSWORD}</sasl_password>
        <ssl_ca_location>${TLS_DIR}/ca.crt</ssl_ca_location>
        <ssl_certificate_location>${TLS_DIR}/client.crt</ssl_certificate_location>
        <ssl_key_location>${TLS_DIR}/client.key</ssl_key_location>
        <ssl_endpoint_identification_algorithm>https</ssl_endpoint_identification_algorithm>
        <enable_ssl_certificate_verification>true</enable_ssl_certificate_verification>
    </kafka>
</clickhouse>
EOF

exec /entrypoint.sh "$@"
