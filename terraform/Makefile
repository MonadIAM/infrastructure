ifndef ROOT
$(error ROOT is required)
endif

STATE_bootstrap = $(VAULT_STATE_DIR)/terraform-bootstrap
STATE_vault-server = $(VAULT_STATE_DIR)/terraform-vault-server
STATE_vault = $(VAULT_STATE_DIR)/terraform

export TF_VAR_kubeconfig = $(KUBECONFIG)
export TF_VAR_context = $(KUBE_CONTEXT)
export TF_VAR_cert_manager_chart = $(CERT_MANAGER_CHART)
export TF_VAR_cert_manager_version = $(CERT_MANAGER_VERSION)
export TF_VAR_vault_chart_version = $(VAULT_CHART_VERSION)
export TF_VAR_hashicorp_repository = $(HASHICORP_HELM_REPO)
export TF_VAR_kv_mount = $(VAULT_KV_MOUNT)

.DEFAULT_GOAL := plan

export TF_IN_AUTOMATION = 1
export TF_DATA_DIR = $(abspath $(STATE_$(ROOT)))/.terraform

.PHONY: init plan apply

# Terraform
init:
	@umask 077; mkdir -p "$(abspath $(STATE_$(ROOT)))"
	terraform -chdir="terraform/$(ROOT)" init -input=false -no-color -lockfile=readonly -backend-config="path=$(abspath $(STATE_$(ROOT)))/terraform.tfstate"
plan: init
	terraform -chdir="terraform/$(ROOT)" plan -input=false -no-color
apply: init
	terraform -chdir="terraform/$(ROOT)" apply -input=false -auto-approve -no-color
