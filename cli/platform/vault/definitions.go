package vault

import "github.com/MonadIAM/infrastructure/cli/config"

const tokenCommand = `IFS= read -r VAULT_TOKEN; export VAULT_TOKEN; exec "$@"`

func credentialDefinitions(environment config.Environment) []credentialDefinition {
	return []credentialDefinition{
		{
			Path:        "monadiam-data/postgresql/main/superuser",
			UserKey:     "username",
			User:        "postgres",
			PasswordKey: "password",
		},
		{
			Path:        "monadiam-data/postgresql/main/app",
			UserKey:     "username",
			User:        "app",
			PasswordKey: "password",
		},
		{
			Path:        "monadiam-data/postgresql/main/debezium",
			UserKey:     "username",
			User:        "debezium",
			PasswordKey: "password",
		},
		{
			Path:        "monadiam-messaging/kafka/main/admin",
			UserKey:     "username",
			User:        "kafka-main-admin",
			PasswordKey: "password",
		},
		{
			Path:        "clickhouse-operator/operator-credentials",
			UserKey:     "username",
			User:        "clickhouse_operator",
			PasswordKey: "password",
		},
		{
			Path:        "monadiam-data/clickhouse/main/app",
			UserKey:     "username",
			User:        "app",
			PasswordKey: "password_sha256_hex",
			Hash:        true,
		},
		{
			Path:        "monadiam-data/clickhouse/main/intercluster",
			PasswordKey: "secret",
		},
		{
			Path:        "monadiam-data/redis/main/auth",
			PasswordKey: "redis-password",
		},
		{
			Path:        "monadiam-observability/grafana/admin",
			UserKey:     "admin-user",
			User:        environment["GRAFANA_ADMIN_USER"],
			PasswordKey: "admin-password",
		},
	}
}

func roleDefinitions() []roleDefinition {
	return []roleDefinition{
		{
			Namespace: "monadiam-apps",
			PolicyKey: "VAULT_POLICY_NAME",
			RoleKey:   "VAULT_K8S_ROLE",
		},
		{
			Namespace: "monadiam-data",
			PolicyKey: "VAULT_DATA_POLICY_NAME",
			RoleKey:   "VAULT_DATA_K8S_ROLE",
		},
		{
			Namespace: "monadiam-messaging",
			PolicyKey: "VAULT_MESSAGING_POLICY_NAME",
			RoleKey:   "VAULT_MESSAGING_K8S_ROLE",
		},
		{
			Namespace: "monadiam-observability",
			PolicyKey: "VAULT_OBSERVABILITY_POLICY_NAME",
			RoleKey:   "VAULT_OBSERVABILITY_K8S_ROLE",
		},
		{
			Namespace: "clickhouse-operator",
			PolicyKey: "VAULT_CLICKHOUSE_OPERATOR_POLICY_NAME",
			RoleKey:   "VAULT_CLICKHOUSE_OPERATOR_K8S_ROLE",
		},
	}
}
