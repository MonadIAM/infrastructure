.DEFAULT_GOAL := env

GO ?= go
GINKGO := $(GO) run github.com/onsi/ginkgo/v2/ginkgo
GOLANGCI_LINT := $(GO) run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.14.0
ENV_FILE ?= .env
AWS_FILE ?= aws.csv

.PHONY: env aws build up down restart stop start lint lint-fix env-check kafka-init debezium-init vault-init vault-bootstrap clickhouse-init seaweedfs-init archive-run utest itest

# Infrastructure
env:
	[ -f .env.example ] && cp -f .env.example .env || echo '.env.example not found'
aws:
	$(GO) run . aws import --csv "$(AWS_FILE)" --env-file "$(ENV_FILE)"
build:
	docker compose build
up:
	docker compose up -d
down:
	docker compose down
restart:
	$(MAKE) down
	$(MAKE) up
stop:
	docker compose stop
start:
	docker compose start

# Development
lint:
	$(GOLANGCI_LINT) run
lint-fix:
	gofmt -w main.go cli
env-check:
	$(GO) run . config validate --env-file "$(ENV_FILE)"

# Initialization
kafka-init:
	docker compose run --rm --no-deps kafka-init
debezium-init:
	docker compose run --rm --no-deps debezium-init
vault-init:
	docker compose run --rm --no-deps vault-init-unseal
vault-bootstrap:
	docker compose run --rm --no-deps vault-bootstrap
clickhouse-init:
	docker compose run --rm --no-deps clickhouse-init
seaweedfs-init:
	docker compose run --rm --no-deps seaweedfs-init

# Archive
archive-run:
	docker compose run --rm --no-deps clickhouse-archiver archive run --env-file=

# Test
utest:
	$(GINKGO) --fd --succinct=false --keep-going --race --skip-package=cli/tests/integration ./cli/...
itest:
	$(GINKGO) --fd --succinct=false --keep-going --tags=integration --timeout=15m ./cli/tests/integration/...
