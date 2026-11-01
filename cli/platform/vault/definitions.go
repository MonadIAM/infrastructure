package vault

import (
	"slices"
	"strings"

	"github.com/MonadIAM/infrastructure/cli/config"
)

var runtimeFields = []secretField{
	{
		Name:        "postgresql_username",
		Environment: "_POSTGRES_USER",
		Required:    true,
	},
	{
		Name:        "postgresql_password",
		Environment: "_POSTGRES_PASSWORD",
		Required:    true,
	},
	{
		Name:        "redis_password",
		Environment: "_REDIS_PASSWORD",
		Required:    true,
	},
}

var notificationFields = []secretField{
	{Name: "aws_access_key_id", Environment: "NOTIFICATION_AWS_ACCESS_KEY_ID"},
	{Name: "aws_secret_access_key", Environment: "NOTIFICATION_AWS_SECRET_ACCESS_KEY"},
}

var oauthSecret = secretDefinition{
	Path: "identity-service/oauth",
	Fields: []secretField{
		{Name: "google_client_secret", Environment: "IDENTITY_OAUTH_GOOGLE_CLIENT_SECRET"},
		{Name: "github_client_secret", Environment: "IDENTITY_OAUTH_GITHUB_CLIENT_SECRET"},
	},
}

func secretDefinitions() []secretDefinition {
	var secrets []secretDefinition

	for _, service := range config.Services() {
		prefix := strings.ToUpper(strings.ReplaceAll(service, "-", "_"))
		fields := slices.Clone(runtimeFields)
		for index := range fields {
			fields[index].Environment = prefix + fields[index].Environment
		}

		if service == "notification" {
			fields = append(fields, notificationFields...)
		}

		secrets = append(secrets, secretDefinition{
			Path:   service + "-service/runtime",
			Fields: fields,
		})
	}

	secrets = append(secrets, secretDefinition{
		Path:   oauthSecret.Path,
		Fields: slices.Clone(oauthSecret.Fields),
	})

	return secrets
}
