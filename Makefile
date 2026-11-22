.DEFAULT_GOAL := env

GO ?= go
GOLANGCI_LINT := $(GO) run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.14.0
PROFILE ?= dev
ENV_FILE ?= .env.$(PROFILE)
COMPONENT ?= all
OBSERVABILITY_APPS := application/monadiam-prometheus-stack application/monadiam-loki application/monadiam-otel-collector

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

.PHONY: env start resume stop restart status build lint lint-fix env-check reconcile platform-plan platform-apply vault-status vault-plan vault-apply seaweedfs applications-ready observability-ready ambient-reconcile gitops-credentials gitops-ui terraform-init terraform-plan terraform-apply
.PHONY: zot-start zot-status zot-logs
.PHONY: redis-backup redis-backup-status redis-backup-logs
.PHONY: microk8s-start microk8s-status

# MicroK8s
microk8s-start:
	sudo microk8s start
	sudo microk8s status --wait-ready
	sudo microk8s kubectl wait --for=condition=Ready node/monadiam-$(PROFILE) --timeout=180s
	mkdir -p "$(HOME)/.kube"
	sudo microk8s config > "$(HOME)/.kube/monadiam-microk8s-$(PROFILE).yaml"
	chmod 600 "$(HOME)/.kube/monadiam-microk8s-$(PROFILE).yaml"
	kubectl --kubeconfig "$(HOME)/.kube/monadiam-microk8s-$(PROFILE).yaml" config rename-context microk8s monadiam-microk8s-$(PROFILE)
	@node_ip=$$(sudo microk8s kubectl get node monadiam-$(PROFILE) -o jsonpath='{.status.addresses[?(@.type=="InternalIP")].address}'); \
		kubectl --kubeconfig "$(HOME)/.kube/monadiam-microk8s-$(PROFILE).yaml" config set-cluster microk8s-cluster --server="https://$$node_ip:16443"
	kubectl --kubeconfig "$(HOME)/.kube/monadiam-microk8s-$(PROFILE).yaml" apply -f clusters/$(PROFILE)/infrastructure/system/limits.yaml
	kubectl --kubeconfig "$(HOME)/.kube/monadiam-microk8s-$(PROFILE).yaml" -n kube-system patch deployment coredns --type=strategic \
		--patch-file clusters/$(PROFILE)/infrastructure/system/coredns.yaml
	kubectl --kubeconfig "$(HOME)/.kube/monadiam-microk8s-$(PROFILE).yaml" -n kube-system patch deployment metrics-server --type=strategic \
		--patch-file clusters/$(PROFILE)/infrastructure/system/metrics-server.yaml
	kubectl --kubeconfig "$(HOME)/.kube/monadiam-microk8s-$(PROFILE).yaml" -n kube-system rollout status deployment/coredns --timeout=180s
microk8s-status:
	microk8s status

# Zot
zot-start:
	@set -a; . "$(abspath $(ENV_FILE))"; set +a; \
		kubectl --kubeconfig "$${KUBECONFIG:?}" --context "$${KUBE_CONTEXT:?}" apply -k infrastructure/namespaces/base && \
		kubectl --kubeconfig "$$KUBECONFIG" --context "$$KUBE_CONTEXT" apply -k infrastructure/zot/base && \
		kubectl --kubeconfig "$$KUBECONFIG" --context "$$KUBE_CONTEXT" -n monadiam-data rollout status deployment/monadiam-registry --timeout=300s
zot-status:
	@set -a; . "$(abspath $(ENV_FILE))"; set +a; \
		kubectl --kubeconfig "$${KUBECONFIG:?}" --context "$${KUBE_CONTEXT:?}" -n monadiam-data get deployment,pod,service -l app.kubernetes.io/name=monadiam-registry
zot-logs:
	@set -a; . "$(abspath $(ENV_FILE))"; set +a; \
		kubectl --kubeconfig "$${KUBECONFIG:?}" --context "$${KUBE_CONTEXT:?}" -n monadiam-data logs -f deployment/monadiam-registry

# Redis backups
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
start: env microk8s-start
	$(MAKE) zot-start
	$(MAKE) COMPONENT=storage platform-apply
	$(MAKE) COMPONENT=cert-manager reconcile
	$(MAKE) COMPONENT=metrics-server reconcile
	$(MAKE) COMPONENT=istio reconcile
	$(MAKE) ambient-reconcile
	$(MAKE) COMPONENT=vault reconcile
	$(MAKE) COMPONENT=gateway reconcile
	$(MAKE) COMPONENT=operators reconcile
	$(MAKE) COMPONENT=clusters reconcile
	$(MAKE) COMPONENT=apicurio reconcile
	$(MAKE) COMPONENT=seaweedfs reconcile
	$(MAKE) COMPONENT=observability platform-apply
	$(MAKE) COMPONENT=gitops platform-apply
	$(MAKE) COMPONENT=gitops-apps platform-apply
	$(MAKE) observability-ready
	$(MAKE) applications-ready
ambient-reconcile:
	kubectl --kubeconfig "$(HOME)/.kube/monadiam-microk8s-$(PROFILE).yaml" --context "monadiam-microk8s-$(PROFILE)" -n istio-system rollout restart daemonset/istio-cni-node
	kubectl --kubeconfig "$(HOME)/.kube/monadiam-microk8s-$(PROFILE).yaml" --context "monadiam-microk8s-$(PROFILE)" -n istio-system rollout status daemonset/istio-cni-node --timeout=300s
observability-ready:
	kubectl --kubeconfig "$(HOME)/.kube/monadiam-microk8s-$(PROFILE).yaml" --context "monadiam-microk8s-$(PROFILE)" -n argocd wait --for=create $(OBSERVABILITY_APPS) --timeout=900s
	kubectl --kubeconfig "$(HOME)/.kube/monadiam-microk8s-$(PROFILE).yaml" --context "monadiam-microk8s-$(PROFILE)" -n argocd wait --for=jsonpath='{.status.sync.status}'=Synced $(OBSERVABILITY_APPS) --timeout=900s
	kubectl --kubeconfig "$(HOME)/.kube/monadiam-microk8s-$(PROFILE).yaml" --context "monadiam-microk8s-$(PROFILE)" -n argocd wait --for=jsonpath='{.status.operationState.phase}'=Succeeded $(OBSERVABILITY_APPS) --timeout=900s
	kubectl --kubeconfig "$(HOME)/.kube/monadiam-microk8s-$(PROFILE).yaml" --context "monadiam-microk8s-$(PROFILE)" -n argocd wait --for=jsonpath='{.status.health.status}'=Healthy $(OBSERVABILITY_APPS) --timeout=900s
applications-ready:
	@set -a; . "$(abspath $(ENV_FILE))"; set +a; \
		for service in access-control identity notification organization hr; do \
			kubectl --kubeconfig "$${KUBECONFIG:?}" --context "$${KUBE_CONTEXT:?}" -n monadiam-apps rollout status \
				"deployment/$$service-service" --timeout=900s || exit; \
		done; \
		for port in 4001 4002 4003 4004 4005; do \
			curl --insecure --fail --silent --show-error --location --retry-all-errors --retry 10 --retry-delay 2 \
				"https://$${APPLICATIONS_IP:?}:$$port/docs/" --output /dev/null || exit; \
		done
resume: start
stop:
	sudo microk8s stop
restart:
	$(MAKE) stop
	$(MAKE) start
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
vault-plan:
	$(GO) run . vault exec --env-file "$(ENV_FILE)" -- make ROOT=vault terraform-plan
vault-apply:
	$(GO) run . vault exec --env-file "$(ENV_FILE)" -- make ROOT=vault terraform-apply
seaweedfs:
	$(GO) run . platform reconcile --component seaweedfs --env-file "$(ENV_FILE)"

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
