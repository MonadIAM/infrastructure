package vault

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	infrahttp "github.com/MonadIAM/infrastructure/cli/platform/http"
)

func Initialize(ctx context.Context, settings Settings, output io.Writer) error {
	err := settings.Validate()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, settings.Timeout)
	defer cancel()

	client := infrahttp.New(settings.Address)

	var status sealStatus
	err = infrahttp.Wait(ctx, time.Second, func() error {
		return client.JSON(ctx, http.MethodGet, "/v1/sys/seal-status", nil, &status)
	})
	if err != nil {
		return err
	}

	err = prepareInit(ctx, client, settings, status.Initialized)
	if err != nil {
		return err
	}

	if status.Sealed {
		key, err := readSecret(settings.InitDir, "unseal-key")
		if err != nil {
			return err
		}

		err = client.JSON(ctx, http.MethodPost, "/v1/sys/unseal", map[string]string{"key": key}, &status)
		if err != nil {
			return err
		}

		if status.Sealed {
			return errors.New("vault remains sealed")
		}
	}

	token, err := readSecret(settings.InitDir, "root-token")
	if err != nil {
		return err
	}

	client.Token = token
	err = ensureLocalToken(ctx, client, settings.LocalToken)
	if err != nil {
		return err
	}

	_, err = fmt.Fprintln(output, "Vault is initialized and unsealed.")
	return err
}

func prepareInit(ctx context.Context, client infrahttp.Client, settings Settings, initialized bool) error {
	err := os.MkdirAll(settings.InitDir, 0700)
	if err != nil {
		return err
	}

	err = os.Chmod(settings.InitDir, 0700)
	if err != nil {
		return err
	}

	backup := filepath.Join(settings.InitDir, "init.json")
	if initialized {
		return recoverCredentials(settings.InitDir)
	}

	credentialFiles := []string{"root-token", "unseal-key", "init.json"}

	for _, name := range credentialFiles {
		_, err := os.Stat(filepath.Join(settings.InitDir, name))
		if !errors.Is(err, os.ErrNotExist) {
			return errors.New("vault is uninitialized but initialization files already exist")
		}
	}

	var result initResult
	err = client.JSON(ctx, http.MethodPost, "/v1/sys/init", map[string]int{
		"secret_shares":    1,
		"secret_threshold": 1,
	}, &result)
	if err != nil {
		return err
	}

	data, err := json.Marshal(result)
	if err != nil {
		return err
	}

	err = writeSecret(backup, data)
	if err == nil {
		return saveCredentials(settings.InitDir, result)
	} else {
		return err
	}
}

func saveCredentials(directory string, result initResult) error {
	if len(result.Keys) != 1 || result.Keys[0] == "" || result.RootToken == "" {
		return errors.New("invalid Vault initialization credentials")
	}

	credentials := map[string]string{
		"unseal-key": result.Keys[0],
		"root-token": result.RootToken,
	}

	for name, value := range credentials {
		path := filepath.Join(directory, name)
		existing, err := os.ReadFile(path)
		if err == nil {
			if strings.TrimSpace(string(existing)) != value {
				return errors.New("conflicting saved Vault credentials")
			}

			continue
		}

		if !errors.Is(err, os.ErrNotExist) {
			return err
		}

		err = writeSecret(path, []byte(value+"\n"))
		if err != nil {
			return err
		}
	}

	return nil
}

func writeSecret(path string, data []byte) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0400)
	if err != nil {
		return err
	}

	defer func() { _ = file.Close() }()

	_, err = file.Write(data)
	if err == nil {
		return file.Sync()
	} else {
		return err
	}
}

func readSecret(directory, name string) (string, error) {
	data, err := os.ReadFile(filepath.Join(directory, name))
	if err != nil || len(strings.TrimSpace(string(data))) == 0 {
		return "", fmt.Errorf("vault %s file is missing or empty", name)
	} else {
		return strings.TrimSpace(string(data)), nil
	}
}

func ensureLocalToken(ctx context.Context, client infrahttp.Client, token string) error {
	local := client
	local.Token = token

	var current tokenResponse
	err := local.JSON(ctx, http.MethodGet, "/v1/auth/token/lookup-self", nil, &current)
	if err == nil {
		if !slices.Contains(current.Data.Policies, "root") || current.Data.TTL != 0 {
			return errors.New("existing local Vault token is not a non-expiring root token")
		} else {
			return nil
		}
	}

	if !infrahttp.IsStatus(err, 403) {
		return err
	} else {
		return client.JSON(ctx, http.MethodPost, "/v1/auth/token/create", map[string]any{
			"id":        token,
			"policies":  []string{"root"},
			"no_parent": true,
			"ttl":       "0",
			"renewable": false,
		}, nil)
	}
}

func recoverCredentials(directory string) error {
	_, tokenErr := os.Stat(filepath.Join(directory, "root-token"))
	_, keyErr := os.Stat(filepath.Join(directory, "unseal-key"))
	if tokenErr == nil && keyErr == nil {
		return nil
	}

	data, err := os.ReadFile(filepath.Join(directory, "init.json"))
	if err != nil {
		return errors.New("vault is initialized but its saved initialization credentials are missing")
	}

	var result initResult
	err = json.Unmarshal(data, &result)
	if err == nil {
		return saveCredentials(directory, result)
	} else {
		return errors.New("invalid saved Vault initialization")
	}
}
