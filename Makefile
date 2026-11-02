.DEFAULT_GOAL := env

GO ?= go
GINKGO := $(GO) run github.com/onsi/ginkgo/v2/ginkgo
GOLANGCI_LINT := $(GO) run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.14.0
PROFILE ?= dev
ENV_FILE ?= .env.$(PROFILE)
COMPONENT ?= all
FILTER ?= bootstrap
FOCUS ?=
INTEGRATION_STATE_DIR ?= $(abspath build/integration)
INTEGRATION_TEST := INTEGRATION_STATE_DIR="$(INTEGRATION_STATE_DIR)" $(GINKGO) -v --show-node-events --fail-on-empty --tags=integration --poll-progress-after=30s --poll-progress-interval=30s --output-dir=build

TERRAFORM_STATE := $(TF_STATE_DIR)/$(PROFILE)/$(KUBE_CONTEXT)/$(ROOT)
export TF_DATA_DIR = $(TERRAFORM_STATE)/.terraform
export TF_IN_AUTOMATION = 1
export TF_VAR_kubeconfig = $(KUBECONFIG)
export TF_VAR_context = $(KUBE_CONTEXT)
export TF_VAR_cert_manager_chart = $(CERT_MANAGER_CHART)
export TF_VAR_cert_manager_version = $(CERT_MANAGER_VERSION)
export TF_VAR_vault_chart_version = $(VAULT_CHART_VERSION)
export TF_VAR_hashicorp_repository = $(HASHICORP_HELM_REPO)
export TF_VAR_kv_mount = $(VAULT_KV_MOUNT)

.PHONY: env start resume stop restart recreate status build lint lint-fix env-check reconcile platform-plan platform-apply vault-status vault-plan vault-apply seaweedfs utest itest itest-prepare itest-cleanup itest-slow gitops-credentials gitops-ui terraform-init terraform-plan terraform-apply

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
	gofmt -w main.go cli tests
env-check:
	$(GO) run . config validate --env-file "$(ENV_FILE)"

# Platform
reconcile:
	$(GO) run . platform reconcile --component "$(COMPONENT)" --env-file "$(ENV_FILE)"
vault-status:
	$(GO) run . vault status --env-file "$(ENV_FILE)"
vault-plan:
	$(GO) run . vault exec --env-file "$(ENV_FILE)" -- make ROOT=vault terraform-plan
vault-apply:
	$(GO) run . vault exec --env-file "$(ENV_FILE)" -- make ROOT=vault terraform-apply
seaweedfs:
	$(GO) run . platform reconcile --component seaweedfs --env-file "$(ENV_FILE)"

# Test
utest:
	$(GINKGO) --fd --succinct=false --keep-going --race ./cli/...
itest:
	mkdir -p build
	INTEGRATION_ACTION=check $(INTEGRATION_TEST) --label-filter="($(FILTER)) && !slow && !environment" --focus="$(FOCUS)" --timeout=15m --json-report=integration-report.json ./tests/integration/...
itest-prepare:
	mkdir -p build
	INTEGRATION_ACTION=prepare $(INTEGRATION_TEST) --label-filter=environment-prepare --timeout=65m --json-report=integration-prepare-report.json ./tests/integration/...
itest-cleanup:
	mkdir -p build
	INTEGRATION_ACTION=cleanup $(INTEGRATION_TEST) --label-filter=environment-cleanup --timeout=6m --json-report=integration-cleanup-report.json ./tests/integration/...
itest-slow:
	mkdir -p build
	INTEGRATION_ACTION=check $(INTEGRATION_TEST) --label-filter="($(FILTER)) && slow && !environment" --focus="$(FOCUS)" --timeout=2h --json-report=integration-slow-report.json ./tests/integration/...

platform-plan:
	@set -a; . "$(abspath $(ENV_FILE))"; set +a; $(MAKE) ROOT="$(COMPONENT)" terraform-plan
platform-apply:
	@set -a; . "$(abspath $(ENV_FILE))"; set +a; $(MAKE) ROOT="$(COMPONENT)" terraform-apply

# Terraform
terraform-init:
	@umask 077; mkdir -p "$(TERRAFORM_STATE)"
	terraform -chdir="terraform/$(ROOT)" init -input=false -no-color -lockfile=readonly -backend-config="path=$(TERRAFORM_STATE)/terraform.tfstate"
terraform-plan: terraform-init
	terraform -chdir="terraform/$(ROOT)" plan -input=false -no-color
terraform-apply: terraform-init
	terraform -chdir="terraform/$(ROOT)" apply -input=false -auto-approve -no-color

gitops-credentials:
	@set -a; . "$(abspath $(ENV_FILE))"; set +a; \
	: "$${GITHUB_APP_ID:?}" "$${GITHUB_APP_INSTALLATION_ID:?}" "$${GITHUB_APP_PRIVATE_KEY_FILE:?}" "$${KUBECONFIG:?}" "$${KUBE_CONTEXT:?}"; \
	kubectl --kubeconfig "$${KUBECONFIG}" --context "$${KUBE_CONTEXT}" -n argocd create secret generic monadiam-image-updater-github \
		--from-literal=githubAppID="$${GITHUB_APP_ID}" \
		--from-literal=githubAppInstallationID="$${GITHUB_APP_INSTALLATION_ID}" \
		--from-file=githubAppPrivateKey="$${GITHUB_APP_PRIVATE_KEY_FILE}" \
		--dry-run=client -o yaml | kubectl --kubeconfig "$${KUBECONFIG}" --context "$${KUBE_CONTEXT}" apply --server-side --field-manager=gitops-credentials -f -
gitops-ui:
	@set -a; . "$(abspath $(ENV_FILE))"; set +a; kubectl --kubeconfig "$${KUBECONFIG:?}" --context "$${KUBE_CONTEXT:?}" -n argocd port-forward --address=127.0.0.1 service/argocd-server 8443:443
