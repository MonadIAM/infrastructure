#!/bin/sh
set -eu

CONSUL_HTTP_ADDR="${CONSUL_HTTP_ADDR:-http://consul:8500}"
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

consul intention create -deny -replace "*" "*"
consul intention create -allow -replace "identity-service" "access-control-service"
consul intention create -allow -replace "notification-service" "access-control-service"
consul intention create -allow -replace "template-service" "access-control-service"
