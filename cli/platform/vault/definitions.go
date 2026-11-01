package vault

import (
	"github.com/MonadIAM/infrastructure/cli/config"
)

const tokenCommand = `IFS= read -r VAULT_TOKEN; export VAULT_TOKEN; exec "$@"`

func credentialDefinitions(environment config.Environment) []credentialDefinition {
	return []credentialDefinition{
		{
			Path:        "monadiam-data/postgresql/access-control/app",
			UserKey:     "username",
			User:        "access_control",
			PasswordKey: "password",
		},
		{
			Path:        "monadiam-data/postgresql/access-control/debezium",
			UserKey:     "username",
			User:        "debezium",
			PasswordKey: "password",
		},
		{
			Path:        "monadiam-data/postgresql/identity/app",
			UserKey:     "username",
			User:        "identity",
			PasswordKey: "password",
		},
		{
			Path:        "monadiam-data/postgresql/identity/debezium",
			UserKey:     "username",
			User:        "debezium",
			PasswordKey: "password",
		},
		{
			Path:        "monadiam-data/postgresql/notification/app",
			UserKey:     "username",
			User:        "notification",
			PasswordKey: "password",
		},
		{
			Path:        "monadiam-data/postgresql/notification/debezium",
			UserKey:     "username",
			User:        "debezium",
			PasswordKey: "password",
		},
		{
			Path:        "monadiam-data/postgresql/organization/app",
			UserKey:     "username",
			User:        "organization",
			PasswordKey: "password",
		},
		{
			Path:        "monadiam-data/postgresql/organization/debezium",
			UserKey:     "username",
			User:        "debezium",
			PasswordKey: "password",
		},
		{
			Path:        "monadiam-data/postgresql/hr/app",
			UserKey:     "username",
			User:        "hr",
			PasswordKey: "password",
		},
		{
			Path:        "monadiam-data/postgresql/hr/debezium",
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
			Path:        "monadiam-data/redis/access-control/auth",
			PasswordKey: "redis-password",
		},
		{
			Path:        "monadiam-data/redis/identity/auth",
			PasswordKey: "redis-password",
		},
		{
			Path:        "monadiam-data/redis/notification/auth",
			PasswordKey: "redis-password",
		},
		{
			Path:        "monadiam-data/redis/organization/auth",
			PasswordKey: "redis-password",
		},
		{
			Path:        "monadiam-data/redis/hr/auth",
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
