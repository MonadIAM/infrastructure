package vault

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/MonadIAM/infrastructure/cli/config"
	infrahttp "github.com/MonadIAM/infrastructure/cli/platform/http"
)

func ValidateBootstrap(settings Settings) error {
	err := settings.Validate()
	if err != nil {
		return err
	}

	for _, secret := range secretDefinitions() {
		for _, field := range secret.Fields {
			if field.Required {
				err := settings.Environment.Require(field.Environment)
				if err != nil {
					return err
				}
			}
		}
	}

	_, err = readPolicies(settings.PoliciesDir)
	return err
}

func Bootstrap(ctx context.Context, settings Settings, output io.Writer) error {
	err := ValidateBootstrap(settings)
	if err != nil {
		return err
	}

	policies, err := readPolicies(settings.PoliciesDir)
	if err != nil {
		return err
	}

	token, err := readSecret(settings.InitDir, "root-token")
	if err != nil {
		return err
	}

	client := infrahttp.New(settings.Address)
	client.Token = token

	ctx, cancel := context.WithTimeout(ctx, settings.Timeout)
	defer cancel()

	for _, definition := range mountDefinitions() {
		err := ensureMount(ctx, client, definition)
		if err != nil {
			return err
		}
	}

	err = writeRuntime(ctx, client, settings.Environment)
	if err != nil {
		return err
	}

	for _, key := range keyDefinitions() {
		err := ensureKey(ctx, client, key)
		if err != nil {
			return err
		}
	}

	err = configureAccess(ctx, client, policies)
	if err != nil {
		return err
	}

	_, err = fmt.Fprintln(output, "Vault KV, AppRole and Transit are configured.")
	return err
}

func readPolicies(directory string) (map[string]string, error) {
	policies := make(map[string]string)

	for _, role := range roleDefinitions() {
		for _, name := range role.Policies {
			data, err := os.ReadFile(filepath.Join(directory, name+".hcl"))
			if err != nil {
				return nil, fmt.Errorf("cannot read Vault policy %s", name)
			}

			if strings.TrimSpace(string(data)) == "" {
				return nil, errors.New("vault policy is empty")
			}

			policies[name] = string(data)
		}
	}

	return policies, nil
}

func ensureMount(ctx context.Context, client infrahttp.Client, definition mountDefinition) error {
	var mounts mountList
	err := client.JSON(ctx, http.MethodGet, "/v1/"+definition.Endpoint, nil, &mounts)
	if err != nil {
		return err
	}

	existing, found := mounts.Data[definition.Name+"/"]
	if found {
		if existing.Type != definition.Type {
			return fmt.Errorf("vault mount %s has incompatible type", definition.Name)
		}

		for key, value := range definition.Options {
			if existing.Options[key] != value {
				return fmt.Errorf("vault mount %s has incompatible %s", definition.Name, key)
			}
		}

		return nil
	}

	return client.JSON(ctx, http.MethodPost, "/v1/"+definition.Endpoint+"/"+definition.Name, map[string]any{
		"type":    definition.Type,
		"options": definition.Options,
	}, nil)
}

func writeRuntime(ctx context.Context, client infrahttp.Client, environment config.Environment) error {
	for _, definition := range secretDefinitions() {
		payload := make(map[string]string, len(definition.Fields))

		for _, field := range definition.Fields {
			payload[field.Name] = environment[field.Environment]
		}

		err := client.JSON(ctx, http.MethodPost, "/v1/kv/data/"+definition.Path, map[string]any{"data": payload}, nil)
		if err != nil {
			return err
		}
	}

	return nil
}

func ensureKey(ctx context.Context, client infrahttp.Client, key keyDefinition) error {
	path := "/v1/transit/keys/" + key.Name
	var existing keyResponse
	err := client.JSON(ctx, http.MethodGet, path, nil, &existing)
	if err != nil && !infrahttp.IsStatus(err, 404) {
		return err
	}

	if err == nil && existing.Data.Type != key.Type {
		return fmt.Errorf("vault key %s has incompatible type", key.Name)
	}

	if infrahttp.IsStatus(err, 404) {
		payload := map[string]any{"type": key.Type}
		if key.KeySize > 0 {
			payload["key_size"] = key.KeySize
		}

		err := client.JSON(ctx, http.MethodPost, path, payload, nil)
		if err != nil {
			return err
		}
	}

	if key.Signing {
		return client.JSON(ctx, http.MethodPost, path+"/config", map[string]bool{"deletion_allowed": true}, nil)
	} else {
		return nil
	}
}

func configureAccess(ctx context.Context, client infrahttp.Client, policies map[string]string) error {
	for name, policy := range policies {
		err := client.JSON(ctx, http.MethodPut, "/v1/sys/policies/acl/"+name, map[string]string{"policy": policy}, nil)
		if err != nil {
			return err
		}
	}

	for _, role := range roleDefinitions() {
		err := client.JSON(ctx, http.MethodPost, "/v1/auth/approle/role/"+role.Name, role, nil)
		if err != nil {
			return err
		}
	}

	return nil
}
