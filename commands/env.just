set quiet
set working-directory := '..'

profile := env_var_or_default('PROFILE', 'dev')
env_file := env_var_or_default('ENV_FILE', '.env.' + profile)

# Create the selected profile's environment file if it does not exist.
init:
    go run . config init --env-file '{{ env_file }}'

# Validate an environment file with the Go CLI; default to the active profile.
check file=env_file:
    go run . config validate --env-file '{{ file }}'
