set quiet
set working-directory := '..'

profile := env_var_or_default('PROFILE', 'dev')
env_file := env_var_or_default('ENV_FILE', '.env.' + profile)

# Show Vault status through the Go CLI.
status:
    go run . vault status --env-file '{{ env_file }}'

# Show the Vault Terraform plan with credentials supplied by the Go CLI.
plan:
    go run . vault exec --env-file '{{ env_file }}' -- just tf::plan vault

# Apply Vault Terraform with credentials supplied by the Go CLI.
apply:
    go run . vault exec --env-file '{{ env_file }}' -- just tf::apply vault
