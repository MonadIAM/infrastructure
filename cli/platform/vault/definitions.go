package vault

import "github.com/MonadIAM/infrastructure/cli/config"

const (
	hmacKeySize          = 32
	defaultTokenTTL      = "1h"
	defaultTokenMaxTTL   = "24h"
	unlimitedSecretIDTTL = "0"
)

func mountDefinitions() []mountDefinition {
	return []mountDefinition{
		{
			Endpoint: "sys/mounts",
			Name:     "kv",
			Type:     "kv",
			Options:  map[string]string{"version": "2"},
		},
		{
			Endpoint: "sys/mounts",
			Name:     "transit",
			Type:     "transit",
		},
		{
			Endpoint: "sys/auth",
			Name:     "approle",
			Type:     "approle",
		},
	}
}

func keyDefinitions() []keyDefinition {
	result := []keyDefinition{
		{
			Name:    "identity-access-token",
			Type:    "ecdsa-p256",
			Signing: true,
		},
		{
			Name:    "identity-refresh-token",
			Type:    "ecdsa-p256",
			Signing: true,
		},
		{
			Name:    "identity-id-token",
			Type:    "ecdsa-p256",
			Signing: true,
		},
		{
			Name: "identity-totp-secret",
			Type: "aes256-gcm96",
		},
		{
			Name:    "identity-pairwise-sub",
			Type:    "hmac",
			KeySize: hmacKeySize,
		},
	}

	for _, service := range config.Services() {
		result = append(result, keyDefinition{
			Name:    service + "-audit-mask-hmac",
			Type:    "hmac",
			KeySize: hmacKeySize,
		})
		result = append(result, keyDefinition{
			Name:    service + "-audit-log-sig",
			Type:    "ed25519",
			Signing: true,
		})
	}

	return result
}

func roleDefinitions() []roleDefinition {
	var roles []roleDefinition

	for _, service := range config.Services() {
		name := service + "-service"
		roles = append(roles, roleDefinition{
			Name:            name,
			Policies:        []string{name, name + "-transit"},
			TokenTTL:        defaultTokenTTL,
			TokenMaxTTL:     defaultTokenMaxTTL,
			SecretIDTTL:     unlimitedSecretIDTTL,
			SecretIDNumUses: 0,
		})
	}

	return roles
}

func secretDefinitions() []secretDefinition {
	var secrets []secretDefinition

	for _, service := range config.Services() {
		prefix := config.Prefix(service)
		fields := []secretField{
			{
				Name:        "postgresql_username",
				Environment: prefix + "_POSTGRES_USER",
				Required:    true,
			},
			{
				Name:        "postgresql_password",
				Environment: prefix + "_POSTGRES_PASSWORD",
				Required:    true,
			},
			{
				Name:        "redis_password",
				Environment: prefix + "_REDIS_PASSWORD",
				Required:    true,
			},
		}

		if service == "notification" {
			fields = append(fields,
				secretField{
					Name:        "aws_access_key_id",
					Environment: "NOTIFICATION_AWS_ACCESS_KEY_ID",
				},
				secretField{
					Name:        "aws_secret_access_key",
					Environment: "NOTIFICATION_AWS_SECRET_ACCESS_KEY",
				},
			)
		}

		secrets = append(secrets, secretDefinition{
			Path:   service + "-service/runtime",
			Fields: fields,
		})
	}

	secrets = append(secrets, secretDefinition{
		Path: "identity-service/oauth",
		Fields: []secretField{
			{
				Name:        "google_client_secret",
				Environment: "IDENTITY_OAUTH_GOOGLE_CLIENT_SECRET",
			},
			{
				Name:        "github_client_secret",
				Environment: "IDENTITY_OAUTH_GITHUB_CLIENT_SECRET",
			},
		},
	})

	return secrets
}
