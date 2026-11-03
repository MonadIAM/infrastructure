.DEFAULT_GOAL := env

GO ?= go
GINKGO := $(GO) run github.com/onsi/ginkgo/v2/ginkgo
GOLANGCI_LINT := $(GO) run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.14.0
ENV_FILE ?= .env
AWS_FILE ?= aws.csv

export TF_VAR_policies_dir = $(abspath $(VAULT_POLICIES_DIR))
export TF_IN_AUTOMATION = 1
export TF_DATA_DIR = $(abspath $(VAULT_INIT_DIR)/terraform)/.terraform

.PHONY: env aws build up down restart stop start lint lint-fix env-check kafka-init debezium-init vault-init vault-bootstrap vault-plan terraform-init terraform-plan terraform-apply clickhouse-init seaweedfs-init archive-run utest itest

# Infrastructure
env:
	cp .env.example "$(ENV_FILE)"
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
	gofmt -w main.go cli tests
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
vault-plan:
	docker compose run --rm --no-deps --entrypoint infra vault-bootstrap vault exec --env-file= -- make terraform-plan
clickhouse-init:
	docker compose run --rm --no-deps clickhouse-init
seaweedfs-init:
	docker compose run --rm --no-deps seaweedfs-init

# Terraform
terraform-init:
	@umask 077; mkdir -p "$(abspath $(VAULT_INIT_DIR)/terraform)"
	terraform -chdir=terraform/vault init -input=false -no-color -lockfile=readonly -backend-config="path=$(abspath $(VAULT_INIT_DIR)/terraform)/terraform.tfstate"
terraform-plan: terraform-init
	terraform -chdir=terraform/vault plan -input=false -no-color
terraform-apply: terraform-init
	terraform -chdir=terraform/vault apply -input=false -auto-approve -no-color

# Archive
archive-run:
	docker compose run --rm --no-deps clickhouse-archiver archive run --env-file=

# Test
utest:
	$(GINKGO) --fd --succinct=false --keep-going --race ./cli/...
itest:
	$(GINKGO) --fd --succinct=false --keep-going --tags=integration --timeout=15m ./tests/integration/...
