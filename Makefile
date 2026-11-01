.DEFAULT_GOAL := env

GO ?= go
GINKGO := $(GO) run github.com/onsi/ginkgo/v2/ginkgo
GOLANGCI_LINT := $(GO) run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.14.0
ENV_FILE ?= .env
COMPONENT ?= all
FILTER ?=

.PHONY: env start resume stop restart recreate status build lint lint-fix env-check reconcile platform-plan platform-apply vault-status vault-plan vault-apply seaweedfs utest itest gitops-credentials gitops-ui

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
	@set -a; . "$(abspath $(ENV_FILE))"; set +a; $(GO) run . vault exec --env-file= -- make -f terraform/Makefile ROOT=vault plan
vault-apply:
	@set -a; . "$(abspath $(ENV_FILE))"; set +a; $(GO) run . vault exec --env-file= -- make -f terraform/Makefile ROOT=vault apply
seaweedfs:
	$(GO) run . platform reconcile --component seaweedfs --env-file "$(ENV_FILE)"

# Test
utest:
	$(GINKGO) --fd --succinct=false --keep-going --race ./cli/...
itest:
	mkdir -p build
	$(GINKGO) --fd --succinct=false --keep-going --tags=integration --label-filter="$(FILTER)" --timeout=2h --output-dir=build --json-report=integration-report.json ./tests/integration/...

platform-plan:
	@set -a; . "$(abspath $(ENV_FILE))"; set +a; $(MAKE) -f terraform/Makefile ROOT="$(COMPONENT)" plan
platform-apply:
	@set -a; . "$(abspath $(ENV_FILE))"; set +a; $(MAKE) -f terraform/Makefile ROOT="$(COMPONENT)" apply

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
