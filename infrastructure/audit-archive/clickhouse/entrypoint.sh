#!/bin/sh
set -eu

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

export CLICKHOUSE_PASSWORD="$(cat /vault/secrets/clickhouse_password)"
KAFKA_SASL_PASSWORD="$(tr -d '\r\n' < /vault/secrets/kafka_sasl_password)"

mkdir -p /etc/clickhouse-server/config.d

cat > /etc/clickhouse-server/config.d/kafka.xml <<EOF
<clickhouse>
    <kafka>
        <security_protocol>sasl_ssl</security_protocol>
        <sasl_mechanism>SCRAM-SHA-512</sasl_mechanism>
        <sasl_username>clickhouse</sasl_username>
        <sasl_password>${KAFKA_SASL_PASSWORD}</sasl_password>
        <ssl_ca_location>/tls/ca.crt</ssl_ca_location>
        <ssl_certificate_location>/tls/client.crt</ssl_certificate_location>
        <ssl_key_location>/tls/client.key</ssl_key_location>
        <ssl_endpoint_identification_algorithm>https</ssl_endpoint_identification_algorithm>
        <enable_ssl_certificate_verification>true</enable_ssl_certificate_verification>
    </kafka>
</clickhouse>
EOF

exec /entrypoint.sh "$@"
