package vault

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

func (client Client) EnsureCredentials(ctx context.Context) error {
	for _, value := range credentialDefinitions(client.Environment) {
		err := client.ensureCredential(ctx, value)
		if err != nil {
			return err
		}
	}

	return client.ensureSeaweedFS(ctx)
}

func (client Client) get(ctx context.Context, path string) (map[string]string, int, error) {
	result, err := client.authenticated(ctx, nil, "kv", "get", "-format=json", client.Environment["VAULT_KV_MOUNT"]+"/"+path)
	if err != nil {
		if strings.HasPrefix(strings.TrimSpace(string(result.Stderr)), "No value found at ") ||
			strings.HasPrefix(strings.TrimSpace(string(result.Output)), "No value found at ") {
			return nil, 0, nil
		}

		return nil, 0, fmt.Errorf("reading Vault secret %s: %w", path, err)
	}

	var response secretResponse
	err = json.Unmarshal(result.Output, &response)
	if err != nil || response.Data.Data == nil || response.Data.Metadata.Version < 1 {
		return nil, 0, errors.New("invalid Vault KV response")
	}

	return response.Data.Data, response.Data.Metadata.Version, nil
}

func (client Client) ensureCredential(ctx context.Context, value credentialDefinition) error {
	current, version, err := client.get(ctx, value.Path)
	if err != nil {
		return err
	}

	if current != nil {
		if value.Path != "clickhouse-operator/operator-credentials" || urlSafe(current["password"]) {
			return nil
		}
	}

	random := make([]byte, 32)
	_, err = rand.Read(random)
	if err != nil {
		return err
	}

	password := hex.EncodeToString(random)
	if value.Hash {
		sum := sha256.Sum256([]byte(password))
		password = hex.EncodeToString(sum[:])
	}

	data := map[string]string{value.PasswordKey: password}
	if value.UserKey != "" {
		data[value.UserKey] = value.User
	}

	return client.put(ctx, value.Path, data, fmt.Sprint(version))
}

func urlSafe(value string) bool {
	if value == "" {
		return false
	}

	for _, char := range value {
		if char >= 'a' && char <= 'z' || char >= 'A' && char <= 'Z' || char >= '0' && char <= '9' || strings.ContainsRune("._~-", char) {
			continue
		}

		return false
	}

	return true
}
