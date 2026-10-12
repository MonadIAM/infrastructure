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

	"github.com/go-resty/resty/v2"

	"github.com/MonadIAM/infrastructure/cli/config"
	infrahttp "github.com/MonadIAM/infrastructure/cli/platform/http"
)

type Service struct {
	settings Settings
	client   *resty.Client
}

func New(settings Settings) *Service {
	return &Service{settings: settings, client: infrahttp.New(settings.Address)}
}

func (service *Service) Initialize(ctx context.Context, output io.Writer) error {
	err := service.settings.Validate()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, service.settings.Timeout)
	defer cancel()

	status, err := service.waitUntilReady(ctx)
	if err != nil {
		return err
	}

	err = service.prepareInit(ctx, status.Initialized)
	if err != nil {
		return err
	}

	if status.Sealed {
		key, err := readSecret(service.settings.InitDir, "unseal-key")
		if err != nil {
			return err
		}

		response, err := service.client.R().
			SetContext(ctx).
			SetBody(map[string]string{"key": key}).
			ForceContentType("application/json").
			SetResult(&status).
			Post("/v1/sys/unseal")
		if err != nil {
			return fmt.Errorf("vault.Service.Initialize: unseal: %w", infrahttp.RequestError{Cause: err})
		}

		if !response.IsSuccess() {
			return fmt.Errorf("vault.Service.Initialize: unseal: HTTP status %d", response.StatusCode())
		}

		if status.Sealed {
			return errors.New("vault remains sealed")
		}
	}

	token, err := readSecret(service.settings.InitDir, "root-token")
	if err != nil {
		return err
	}

	service.client.SetHeader("X-Vault-Token", token)
	err = service.ensureLocalToken(ctx)
	if err != nil {
		return err
	}

	_, err = fmt.Fprintln(output, "Vault is initialized and unsealed.")
	return err
}

func (service *Service) ValidateBootstrap() error {
	err := service.settings.Validate()
	if err != nil {
		return err
	}

	for _, secret := range secretDefinitions() {
		for _, field := range secret.Fields {
			if field.Required {
				err := service.settings.Environment.Require(field.Environment)
				if err != nil {
					return err
				}
			}
		}
	}

	_, err = service.readPolicies()
	return err
}

func (service *Service) Bootstrap(ctx context.Context, output io.Writer) error {
	err := service.ValidateBootstrap()
	if err != nil {
		return err
	}

	policies, err := service.readPolicies()
	if err != nil {
		return err
	}

	token, err := readSecret(service.settings.InitDir, "root-token")
	if err != nil {
		return err
	}

	service.client.SetHeader("X-Vault-Token", token)

	ctx, cancel := context.WithTimeout(ctx, service.settings.Timeout)
	defer cancel()

	for _, definition := range mountDefinitions {
		err := service.ensureMount(ctx, definition)
		if err != nil {
			return err
		}
	}

	err = service.writeRuntime(ctx)
	if err != nil {
		return err
	}

	for _, key := range keyDefinitions() {
		err := service.ensureKey(ctx, key)
		if err != nil {
			return err
		}
	}

	err = service.configureAccess(ctx, policies)
	if err != nil {
		return err
	}

	_, err = fmt.Fprintln(output, "Vault KV, AppRole and Transit are configured.")
	return err
}

func (service *Service) waitUntilReady(ctx context.Context) (sealStatus, error) {
	client := infrahttp.New(service.settings.Address).
		SetRetryCount(int(service.settings.Timeout/time.Second) + 1).
		SetRetryWaitTime(time.Second).
		SetRetryMaxWaitTime(time.Second)

	var status sealStatus
	response, err := client.R().
		SetContext(ctx).
		ForceContentType("application/json").
		SetResult(&status).
		AddRetryCondition(func(response *resty.Response, err error) bool {
			if response != nil && response.StatusCode() != 0 {
				return response.StatusCode() == 429 || response.StatusCode() >= 500
			} else {
				return err != nil
			}
		}).
		Get("/v1/sys/seal-status")
	if err != nil {
		return status, fmt.Errorf("vault.Service.waitUntilReady: %w", infrahttp.RequestError{Cause: err})
	}

	if response.IsSuccess() {
		return status, nil
	} else {
		return status, fmt.Errorf("vault.Service.waitUntilReady: HTTP status %d", response.StatusCode())
	}
}

func (service *Service) prepareInit(ctx context.Context, initialized bool) error {
	err := os.MkdirAll(service.settings.InitDir, 0700)
	if err != nil {
		return err
	}

	err = os.Chmod(service.settings.InitDir, 0700)
	if err != nil {
		return err
	}

	backup := filepath.Join(service.settings.InitDir, "init.json")
	if initialized {
		return service.recoverCredentials()
	}

	credentialFiles := []string{"root-token", "unseal-key", "init.json"}

	for _, name := range credentialFiles {
		_, err := os.Stat(filepath.Join(service.settings.InitDir, name))
		if !errors.Is(err, os.ErrNotExist) {
			return errors.New("vault is uninitialized but initialization files already exist")
		}
	}

	var result initResult
	response, err := service.client.R().
		SetContext(ctx).
		SetBody(map[string]int{"secret_shares": 1, "secret_threshold": 1}).
		ForceContentType("application/json").
		SetResult(&result).
		Post("/v1/sys/init")
	if err != nil {
		return fmt.Errorf("vault.Service.prepareInit: %w", infrahttp.RequestError{Cause: err})
	}

	if !response.IsSuccess() {
		return fmt.Errorf("vault.Service.prepareInit: HTTP status %d", response.StatusCode())
	}

	data, err := json.Marshal(result)
	if err != nil {
		return err
	}

	err = writeSecret(backup, data)
	if err == nil {
		return service.saveCredentials(result)
	} else {
		return err
	}
}

func (service *Service) ensureLocalToken(ctx context.Context) error {
	var current tokenResponse
	response, err := service.client.R().
		SetContext(ctx).
		SetHeader("X-Vault-Token", service.settings.LocalToken).
		ForceContentType("application/json").
		SetResult(&current).
		Get("/v1/auth/token/lookup-self")
	if err != nil {
		return fmt.Errorf("vault.Service.ensureLocalToken: lookup: %w", infrahttp.RequestError{Cause: err})
	}

	if response.IsSuccess() {
		if !slices.Contains(current.Data.Policies, "root") || current.Data.TTL != 0 {
			return errors.New("existing local Vault token is not a non-expiring root token")
		} else {
			return nil
		}
	}

	if response.StatusCode() != http.StatusForbidden {
		return fmt.Errorf("vault.Service.ensureLocalToken: lookup: HTTP status %d", response.StatusCode())
	}

	response, err = service.client.R().
		SetContext(ctx).
		SetBody(map[string]any{
			"id":        service.settings.LocalToken,
			"policies":  []string{"root"},
			"no_parent": true,
			"ttl":       "0",
			"renewable": false,
		}).
		Post("/v1/auth/token/create")
	if err != nil {
		return fmt.Errorf("vault.Service.ensureLocalToken: create: %w", infrahttp.RequestError{Cause: err})
	}

	if response.IsSuccess() {
		return nil
	} else {
		return fmt.Errorf("vault.Service.ensureLocalToken: create: HTTP status %d", response.StatusCode())
	}
}

func (service *Service) saveCredentials(result initResult) error {
	if len(result.Keys) != 1 || result.Keys[0] == "" || result.RootToken == "" {
		return errors.New("invalid Vault initialization credentials")
	}

	credentials := map[string]string{
		"unseal-key": result.Keys[0],
		"root-token": result.RootToken,
	}

	for name, value := range credentials {
		path := filepath.Join(service.settings.InitDir, name)
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

func (service *Service) recoverCredentials() error {
	_, tokenErr := os.Stat(filepath.Join(service.settings.InitDir, "root-token"))
	_, keyErr := os.Stat(filepath.Join(service.settings.InitDir, "unseal-key"))
	if tokenErr == nil && keyErr == nil {
		return nil
	}

	data, err := os.ReadFile(filepath.Join(service.settings.InitDir, "init.json"))
	if err != nil {
		return errors.New("vault is initialized but its saved initialization credentials are missing")
	}

	var result initResult
	err = json.Unmarshal(data, &result)
	if err == nil {
		return service.saveCredentials(result)
	} else {
		return errors.New("invalid saved Vault initialization")
	}
}

func (service *Service) readPolicies() (map[string]string, error) {
	policies := make(map[string]string)

	for _, role := range roleDefinitions() {
		for _, name := range role.Policies {
			data, err := os.ReadFile(filepath.Join(service.settings.PoliciesDir, name+".hcl"))
			if err != nil {
				return nil, fmt.Errorf("vault.Service.readPolicies: policy %s: %w", name, err)
			}

			if strings.TrimSpace(string(data)) == "" {
				return nil, errors.New("vault policy is empty")
			}

			policies[name] = string(data)
		}
	}

	return policies, nil
}

func (service *Service) ensureMount(ctx context.Context, definition mountDefinition) error {
	var mounts mountList
	response, err := service.client.R().
		SetContext(ctx).
		ForceContentType("application/json").
		SetResult(&mounts).
		Get("/v1/" + definition.Endpoint)
	if err != nil {
		return fmt.Errorf("vault.Service.ensureMount: list: %w", infrahttp.RequestError{Cause: err})
	}

	if !response.IsSuccess() {
		return fmt.Errorf("vault.Service.ensureMount: list: HTTP status %d", response.StatusCode())
	}

	existing, found := mounts.Data[definition.Name+"/"]
	if found {
		if existing.Type != definition.Type {
			return fmt.Errorf("vault.Service.ensureMount: vault mount %s has incompatible type", definition.Name)
		}

		for key, value := range definition.Options {
			if existing.Options[key] != value {
				return fmt.Errorf("vault.Service.ensureMount: vault mount %s has incompatible %s", definition.Name, key)
			}
		}

		return nil
	}

	response, err = service.client.R().
		SetContext(ctx).
		SetBody(map[string]any{"type": definition.Type, "options": definition.Options}).
		Post("/v1/" + definition.Endpoint + "/" + definition.Name)
	if err != nil {
		return fmt.Errorf("vault.Service.ensureMount: create %s: %w", definition.Name, infrahttp.RequestError{Cause: err})
	}

	if response.IsSuccess() {
		return nil
	} else {
		return fmt.Errorf("vault.Service.ensureMount: create %s: HTTP status %d", definition.Name, response.StatusCode())
	}
}

func (service *Service) writeRuntime(ctx context.Context) error {
	for _, definition := range secretDefinitions() {
		payload := make(map[string]string, len(definition.Fields))

		for _, field := range definition.Fields {
			payload[field.Name] = service.settings.Environment.String(field.Environment)
		}

		response, err := service.client.R().
			SetContext(ctx).
			SetBody(map[string]any{"data": payload}).
			Post("/v1/kv/data/" + definition.Path)
		if err != nil {
			return fmt.Errorf("vault.Service.writeRuntime: %s: %w", definition.Path, infrahttp.RequestError{Cause: err})
		}

		if !response.IsSuccess() {
			return fmt.Errorf("vault.Service.writeRuntime: %s: HTTP status %d", definition.Path, response.StatusCode())
		}
	}

	return nil
}

func (service *Service) ensureKey(ctx context.Context, key keyDefinition) error {
	path := "/v1/transit/keys/" + key.Name
	var existing keyResponse
	response, err := service.client.R().
		SetContext(ctx).
		ForceContentType("application/json").
		SetResult(&existing).
		Get(path)
	if err != nil {
		return fmt.Errorf("vault.Service.ensureKey: read %s: %w", key.Name, infrahttp.RequestError{Cause: err})
	}

	switch {
	case response.StatusCode() == http.StatusNotFound:
		payload := map[string]any{"type": key.Type}
		if key.KeySize > 0 {
			payload["key_size"] = key.KeySize
		}

		response, err = service.client.R().SetContext(ctx).SetBody(payload).Post(path)
		if err != nil {
			return fmt.Errorf("vault.Service.ensureKey: create %s: %w", key.Name, infrahttp.RequestError{Cause: err})
		}

		if !response.IsSuccess() {
			return fmt.Errorf("vault.Service.ensureKey: create %s: HTTP status %d", key.Name, response.StatusCode())
		}
	case !response.IsSuccess():
		return fmt.Errorf("vault.Service.ensureKey: read %s: HTTP status %d", key.Name, response.StatusCode())
	case existing.Data.Type != key.Type:
		return fmt.Errorf("vault.Service.ensureKey: vault key %s has incompatible type", key.Name)
	}

	if !key.Signing {
		return nil
	}

	response, err = service.client.R().
		SetContext(ctx).
		SetBody(map[string]bool{"deletion_allowed": true}).
		Post(path + "/config")
	if err != nil {
		return fmt.Errorf("vault.Service.ensureKey: configure %s: %w", key.Name, infrahttp.RequestError{Cause: err})
	}

	if response.IsSuccess() {
		return nil
	} else {
		return fmt.Errorf("vault.Service.ensureKey: configure %s: HTTP status %d", key.Name, response.StatusCode())
	}
}

func (service *Service) configureAccess(ctx context.Context, policies map[string]string) error {
	for name, policy := range policies {
		response, err := service.client.R().
			SetContext(ctx).
			SetBody(map[string]string{"policy": policy}).
			Put("/v1/sys/policies/acl/" + name)
		if err != nil {
			return fmt.Errorf("vault.Service.configureAccess: policy %s: %w", name, infrahttp.RequestError{Cause: err})
		}

		if !response.IsSuccess() {
			return fmt.Errorf("vault.Service.configureAccess: policy %s: HTTP status %d", name, response.StatusCode())
		}
	}

	for _, role := range roleDefinitions() {
		response, err := service.client.R().SetContext(ctx).SetBody(role).Post("/v1/auth/approle/role/" + role.Name)
		if err != nil {
			return fmt.Errorf("vault.Service.configureAccess: role %s: %w", role.Name, infrahttp.RequestError{Cause: err})
		}

		if !response.IsSuccess() {
			return fmt.Errorf("vault.Service.configureAccess: role %s: HTTP status %d", role.Name, response.StatusCode())
		}
	}

	return nil
}

func Load(environment config.Environment) (Settings, error) {
	settings := Settings{
		Address:     "http://vault:8200",
		InitDir:     "/vault/init",
		PoliciesDir: "vault/policies",
		Timeout:     2 * time.Minute,
		Environment: environment,
	}

	err := environment.Decode(&settings)
	if err != nil {
		return settings, err
	}

	return settings, settings.Validate()
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
	if err != nil {
		return "", fmt.Errorf("vault.readSecret: %s: %w", name, err)
	}

	if len(strings.TrimSpace(string(data))) == 0 {
		return "", fmt.Errorf("vault.readSecret: %s file is empty", name)
	} else {
		return strings.TrimSpace(string(data)), nil
	}
}
