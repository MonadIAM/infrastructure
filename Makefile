.DEFAULT_GOAL := env

GO ?= go
GINKGO := $(GO) run github.com/onsi/ginkgo/v2/ginkgo
GOLANGCI_LINT := $(GO) run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.14.0
ENV_FILE ?= .env
COMPONENT ?= all

.PHONY: env start resume stop restart recreate status build lint lint-fix env-check reconcile vault-status seaweedfs utest itest

# Infrastructure
env:
	$(GO) run . config init --env-file "$(ENV_FILE)"
start: env
	$(GO) run . cluster start --env-file "$(ENV_FILE)"
resume: env
	$(GO) run . cluster resume --env-file "$(ENV_FILE)"
stop:
	$(GO) run . cluster stop --env-file "$(ENV_FILE)"
restart: env
	$(GO) run . cluster restart --env-file "$(ENV_FILE)"
recreate: env
	$(GO) run . cluster recreate --confirm-delete --env-file "$(ENV_FILE)"
status:
	$(GO) run . cluster status --env-file "$(ENV_FILE)"

# Development
build:
	$(GO) build -o build/infra .
lint:
	$(GOLANGCI_LINT) run
lint-fix:
	gofmt -w main.go cli
env-check:
	$(GO) run . config validate --env-file "$(ENV_FILE)"

# Platform
reconcile:
	$(GO) run . platform reconcile --component "$(COMPONENT)" --env-file "$(ENV_FILE)"
vault-status:
	$(GO) run . vault status --env-file "$(ENV_FILE)"
seaweedfs:
	$(GO) run . platform reconcile --component seaweedfs --env-file "$(ENV_FILE)"

# Test
utest:
	$(GINKGO) --fd --succinct=false --keep-going --race --skip-package=cli/tests/integration ./cli/...
itest:
	$(GINKGO) --fd --succinct=false --keep-going --tags=integration --timeout=2h --json-report=build/integration-report.json ./cli/tests/integration/...
