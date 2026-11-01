ifndef ROOT
$(error ROOT is required)
endif

STATE_vault = $(VAULT_INIT_DIR)/terraform
STATE_vault-server = $(VAULT_SERVER_STATE_DIR)

export TF_VAR_image = $(VAULT_IMAGE)
export TF_VAR_port = $(VAULT_PUBLISHED_PORT)
export TF_VAR_network = $(DOCKER_NETWORK_VAULT)
export TF_VAR_project = $(COMPOSE_PROJECT_NAME)
export TF_VAR_container_name = $(VAULT_CONTAINER_NAME)
export TF_VAR_config_path = $(abspath vault/config/vault.hcl)
export TF_VAR_policies_dir = $(abspath $(VAULT_POLICIES_DIR))
ifeq ($(ROOT),vault-server)
export TF_VAR_docker_host := $(shell docker context inspect "$(DOCKER_CONTEXT)" --format '{{.Endpoints.docker.Host}}')
endif

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
