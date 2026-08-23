#!/bin/sh
set -eu

: "${ARCHIVE_INTERVAL_SECONDS:?}"

INTERVAL="$ARCHIVE_INTERVAL_SECONDS"

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

echo "clickhouse-archiver starting, will run every ${INTERVAL}s"

while true; do
    bash /archive.sh || echo "archive.sh failed, will retry next cycle" >&2
    sleep "$INTERVAL"
done
