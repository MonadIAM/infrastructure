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
export TF_VAR_profile = $(PROFILE)
export TF_VAR_cert_manager_chart = $(CERT_MANAGER_CHART)
export TF_VAR_cert_manager_version = $(CERT_MANAGER_VERSION)
export TF_VAR_vault_chart_version = $(VAULT_CHART_VERSION)
export TF_VAR_hashicorp_repository = $(HASHICORP_HELM_REPO)
export TF_VAR_kv_mount = $(VAULT_KV_MOUNT)

.PHONY: env start resume stop restart recreate status build lint lint-fix env-check reconcile platform-plan platform-apply vault-status vault-plan vault-apply seaweedfs utest itest itest-prepare itest-cleanup itest-slow gitops-credentials gitops-ui terraform-init terraform-plan terraform-apply
.PHONY: zot-install zot-start zot-stop zot-status zot-logs
.PHONY: redis-backup-install redis-backup redis-backup-status redis-backup-logs
.PHONY: k3s-install k3s-start k3s-status

# K3s
k3s-install: env build/k3s-install.sh /var/lib/monadiam/k3s.ext4 /var/lib/monadiam/k3s-volumes.ext4 zot-install
	sudo install -d /etc/rancher/k3s /etc/systemd/system/k3s.service.d
	sudo install -m 644 bootstrap/k3s/*.mount /etc/systemd/system/
	sudo install -m 644 bootstrap/k3s/resources.conf /etc/systemd/system/k3s.service.d/resources.conf
	sudo install -m 644 bootstrap/k3s/journald.conf /etc/systemd/journald@monadiam.conf
	sudo install -m 600 clusters/$(PROFILE)/k3s/config.yaml /etc/rancher/k3s/config.yaml
	sudo install -m 600 bootstrap/registries.yaml /etc/rancher/k3s/registries.yaml
	sudo systemctl daemon-reload
	sudo systemctl start var-lib-monadiam-k3s-volumes.mount
	sudo install -d /var/lib/monadiam/k3s/data/agent/etc/kubelet.conf.d
	sudo install -m 644 clusters/$(PROFILE)/k3s/kubelet.conf /var/lib/monadiam/k3s/data/agent/etc/kubelet.conf.d/10-monadiam.conf
	@set -a; . "$(abspath $(ENV_FILE))"; set +a; \
		sudo env INSTALL_K3S_VERSION="$${K3S_VERSION:?}" INSTALL_K3S_SKIP_START=true INSTALL_K3S_SKIP_ENABLE=true sh build/k3s-install.sh
build/k3s-install.sh: $(ENV_FILE)
	mkdir -p build
	@set -a; . "$(abspath $(ENV_FILE))"; set +a; \
		curl --fail --location "https://raw.githubusercontent.com/k3s-io/k3s/$${K3S_VERSION:?}/install.sh" --output "$@"
/var/lib/monadiam/k3s.ext4:
	sudo install -d /var/lib/monadiam
	@set -a; . "$(abspath $(ENV_FILE))"; set +a; \
		sudo mkfs.ext4 -m 0 "$@" "$${K3S_STATE_SIZE:?}"
/var/lib/monadiam/k3s-volumes.ext4:
	sudo install -d /var/lib/monadiam
	@set -a; . "$(abspath $(ENV_FILE))"; set +a; \
		sudo mkfs.ext4 -m 0 "$@" "$${K3S_VOLUMES_SIZE:?}"
k3s-start:
	sudo systemctl start k3s
	sudo k3s kubectl wait --for=create node/monadiam-$(PROFILE) --timeout=180s
	sudo k3s kubectl wait --for=condition=Ready node --all --timeout=180s
	mkdir -p "$(HOME)/.kube"
	sudo install -m 600 -o "$$(id -u)" -g "$$(id -g)" /etc/rancher/k3s/k3s.yaml "$(HOME)/.kube/monadiam-$(PROFILE).yaml"
	kubectl --kubeconfig "$(HOME)/.kube/monadiam-$(PROFILE).yaml" config rename-context default monadiam-$(PROFILE)
	kubectl --kubeconfig "$(HOME)/.kube/monadiam-$(PROFILE).yaml" apply --server-side --field-manager=resource-budget \
		-f clusters/$(PROFILE)/system/limits.yaml -f clusters/$(PROFILE)/system/coredns.yaml -f clusters/$(PROFILE)/system/metrics-server.yaml
k3s-status:
	systemctl status k3s --no-pager
	systemctl show monadiam.slice -p CPUQuotaPerSecUSec -p MemoryHigh -p MemoryMax -p MemoryCurrent

# Zot
zot-install: /var/lib/monadiam/zot.ext4
	sudo install -d /etc/monadiam/zot
	sudo install -m 644 clusters/$(PROFILE)/k3s/monadiam.slice /etc/systemd/system/
	sudo install -m 644 bootstrap/zot/compose.yaml bootstrap/zot/config.json /etc/monadiam/zot/
	sudo install -m 644 bootstrap/zot/monadiam-zot.service bootstrap/zot/var-lib-monadiam-zot.mount /etc/systemd/system/
	sudo systemctl daemon-reload
	sudo systemctl enable monadiam-zot.service
	sudo systemctl restart monadiam-zot.service
/var/lib/monadiam/zot.ext4:
	sudo install -d /var/lib/monadiam
	sudo mkfs.ext4 -m 0 /var/lib/monadiam/zot.ext4 32G
zot-start:
	sudo systemctl start monadiam-zot.service
zot-stop:
	sudo systemctl stop monadiam-zot.service
zot-status:
	systemctl status monadiam-zot.service --no-pager
zot-logs:
	journalctl -u monadiam-zot.service -f

# Redis backups
redis-backup-install: /var/lib/monadiam/redis-backups.ext4
	sudo install -d -m 700 /etc/monadiam/redis-backup
	sudo install -m 644 clusters/$(PROFILE)/k3s/monadiam.slice /etc/systemd/system/
	sudo install -m 755 bootstrap/redis-backup/backup.sh /etc/monadiam/redis-backup/
	sudo install -m 644 bootstrap/redis-backup/*.service bootstrap/redis-backup/*.timer bootstrap/redis-backup/*.mount /etc/systemd/system/
	@set -a; . "$(abspath $(ENV_FILE))"; set +a; \
		configuration=$$(mktemp); trap 'rm -f "$$configuration"' EXIT; \
		kubectl --kubeconfig "$${KUBECONFIG:?}" --context "$${KUBE_CONTEXT:?}" config view --minify --flatten --raw > "$$configuration" && \
		sudo install -m 600 "$$configuration" "/etc/monadiam/redis-backup/$(PROFILE).kubeconfig"
	sudo systemctl daemon-reload
	sudo systemctl enable --now monadiam-redis-backup@$(PROFILE).timer
/var/lib/monadiam/redis-backups.ext4:
	sudo install -d /var/lib/monadiam
	sudo mkfs.ext4 -m 0 /var/lib/monadiam/redis-backups.ext4 8G
redis-backup:
	sudo systemctl start monadiam-redis-backup@$(PROFILE).service
redis-backup-status:
	systemctl status monadiam-redis-backup@$(PROFILE).timer --no-pager
	systemctl show monadiam-redis-backup@$(PROFILE).service -p Result -p ExecMainStatus -p MemoryPeak
redis-backup-logs:
	journalctl -u monadiam-redis-backup@$(PROFILE).service -f

# Infrastructure
env:
	$(GO) run . config init --env-file "$(ENV_FILE)"
start: env zot-start
	$(GO) run . cluster start --env-file "$(ENV_FILE)"
resume: env zot-start
	$(GO) run . cluster resume --env-file "$(ENV_FILE)"
stop:
	$(GO) run . cluster stop --env-file "$(ENV_FILE)"
restart: env zot-start
	$(GO) run . cluster restart --env-file "$(ENV_FILE)"
recreate: env zot-start
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
