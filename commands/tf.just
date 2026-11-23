set quiet
set working-directory := '..'
set dotenv-filename := '.env.dev'

# Check formatting and validate every Terraform root.
validate:
    bash scripts/terraform.check.sh

# Initialize one Terraform root with state and providers scoped to the profile and context.
init root:
    @set -eu; \
        state_dir="$TF_STATE_DIR/$PROFILE/$KUBE_CONTEXT/{{ root }}"; \
        umask 077; \
        mkdir -p "$state_dir"; \
        TF_DATA_DIR="$state_dir/.terraform" TF_IN_AUTOMATION=1 \
            terraform -chdir="terraform/{{ root }}" init \
            -input=false -no-color -lockfile=readonly \
            -backend-config="path=$state_dir/terraform.tfstate"

# Show the selected Terraform root's plan without applying it.
plan root:
    just tf::init '{{ root }}'
    @set -eu; \
        export TF_DATA_DIR="$TF_STATE_DIR/$PROFILE/$KUBE_CONTEXT/{{ root }}/.terraform" \
            TF_IN_AUTOMATION=1 \
            TF_VAR_kubeconfig="$KUBECONFIG" \
            TF_VAR_context="$KUBE_CONTEXT" \
            TF_VAR_profile="$PROFILE" \
            TF_VAR_applications_ip="$APPLICATIONS_IP" \
            TF_VAR_cert_manager_chart="$CERT_MANAGER_CHART" \
            TF_VAR_cert_manager_version="$CERT_MANAGER_VERSION" \
            TF_VAR_vault_chart_version="$VAULT_CHART_VERSION" \
            TF_VAR_hashicorp_repository="$HASHICORP_HELM_REPO" \
            TF_VAR_kv_mount="$VAULT_KV_MOUNT"; \
        terraform -chdir="terraform/{{ root }}" plan -input=false -no-color

# Apply the selected Terraform root without an interactive confirmation.
apply root:
    just tf::init '{{ root }}'
    @set -eu; \
        export TF_DATA_DIR="$TF_STATE_DIR/$PROFILE/$KUBE_CONTEXT/{{ root }}/.terraform" \
            TF_IN_AUTOMATION=1 \
            TF_VAR_kubeconfig="$KUBECONFIG" \
            TF_VAR_context="$KUBE_CONTEXT" \
            TF_VAR_profile="$PROFILE" \
            TF_VAR_applications_ip="$APPLICATIONS_IP" \
            TF_VAR_cert_manager_chart="$CERT_MANAGER_CHART" \
            TF_VAR_cert_manager_version="$CERT_MANAGER_VERSION" \
            TF_VAR_vault_chart_version="$VAULT_CHART_VERSION" \
            TF_VAR_hashicorp_repository="$HASHICORP_HELM_REPO" \
            TF_VAR_kv_mount="$VAULT_KV_MOUNT"; \
        terraform -chdir="terraform/{{ root }}" apply \
            -input=false -auto-approve -no-color
