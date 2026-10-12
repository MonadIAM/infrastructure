package vault

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/MonadIAM/infrastructure/cli/config"
	"github.com/MonadIAM/infrastructure/cli/platform/command"
	"github.com/MonadIAM/infrastructure/cli/platform/kubernetes"
	"github.com/MonadIAM/infrastructure/cli/platform/seaweedfs"
)

type Client struct {
	Environment config.Environment
	Kubernetes  kubernetes.Client
	Output      io.Writer
}

func New(environment config.Environment, kubernetesClient kubernetes.Client, output io.Writer) Client {
	return Client{Environment: environment, Kubernetes: kubernetesClient, Output: output}
}

func (client Client) Status(ctx context.Context) (Status, error) {
	err := client.Environment.Require("KUBE_CONTEXT", "VAULT_NAMESPACE", "VAULT_POD", "VAULT_ADDR", "VAULT_CACERT")
	if err != nil {
		return Status{}, err
	}

	result, err := client.execute(ctx, "", nil, "status", "-format=json")
	if err != nil && result.ExitCode != 2 {
		return Status{}, fmt.Errorf("vault.Client.Status: vault status: %w", err)
	}

	var status Status
	err = json.Unmarshal(result.Output, &status)
	if err != nil || !bytes.Contains(result.Output, []byte(`"sealed"`)) {
		return Status{}, errors.New("invalid Vault status response")
	}

	return status, nil
}

func (client Client) Initialize(ctx context.Context) error {
	status, err := client.Status(ctx)
	if err != nil {
		return err
	}

	if status.Initialized {
		return nil
	}

	path := client.Environment["VAULT_INIT_FILE"]
	directories := []string{client.Environment["VAULT_STATE_DIR"], filepath.Dir(path)}

	for _, directory := range directories {
		err := os.MkdirAll(directory, 0700)
		if err != nil {
			return err
		}

		err = os.Chmod(directory, 0700)
		if err != nil {
			return err
		}
	}

	err = backupMaterial(path)
	if err != nil {
		return err
	}

	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}

	defer func() {
		_ = file.Close()
	}()

	result, err := client.execute(ctx, "", nil, "operator", "init", "-key-shares="+client.Environment["VAULT_KEY_SHARES"],
		"-key-threshold="+client.Environment["VAULT_KEY_THRESHOLD"], "-format=json")

	_, writeErr := file.Write(result.Output)
	if writeErr != nil {
		return fmt.Errorf("vault.Client.Initialize: saving Vault init material: %w", writeErr)
	}

	syncErr := file.Sync()
	if syncErr != nil {
		return fmt.Errorf("vault.Client.Initialize: syncing Vault init material: %w", syncErr)
	}

	if err != nil {
		return fmt.Errorf("vault.Client.Initialize: vault initialization failed; inspect the protected init file: %w", err)
	}

	_, err = client.readMaterial()
	if err != nil {
		return err
	}

	_, err = fmt.Fprintln(client.Output, "Vault init material saved to", path)

	return err
}

func (client Client) Unseal(ctx context.Context) error {
	status, err := client.Status(ctx)
	if err != nil {
		return err
	}

	if !status.Sealed {
		return nil
	}

	keys, err := client.readMaterial()
	if err != nil {
		return err
	}

	if status.Threshold < 1 || len(keys.Keys) < status.Threshold {
		return errors.New("insufficient Vault unseal keys")
	}

	for _, key := range keys.Keys[:status.Threshold] {
		input, err := json.Marshal(map[string]string{"key": key})
		if err != nil {
			return err
		}

		_, err = client.execute(ctx, "", input, "write", "sys/unseal", "-")
		if err != nil {
			return fmt.Errorf("vault.Client.Unseal: vault unseal: %w", err)
		}
	}

	status, err = client.Status(ctx)
	if err != nil {
		return err
	}

	if status.Sealed {
		return errors.New("vault remains sealed after submitting the key threshold")
	} else {
		return nil
	}
}

func (client Client) Bootstrap(ctx context.Context) error {
	env := client.Environment

	err := client.ensureMount(ctx, "secrets", env["VAULT_KV_MOUNT"], "-path="+env["VAULT_KV_MOUNT"], "kv-v2")
	if err != nil {
		return err
	}

	err = client.ensureMount(ctx, "auth", env["VAULT_K8S_AUTH_MOUNT"], "-path="+env["VAULT_K8S_AUTH_MOUNT"], "kubernetes")
	if err != nil {
		return err
	}

	_, err = client.authenticated(ctx, nil, "write", "auth/"+env["VAULT_K8S_AUTH_MOUNT"]+"/config", "kubernetes_host=https://kubernetes.default.svc:443")
	if err != nil {
		return err
	}

	for _, binding := range roleDefinitions() {
		err := client.configureRole(ctx, binding.Namespace, env[binding.PolicyKey], env[binding.RoleKey])
		if err != nil {
			return err
		}
	}

	err = client.ensureMount(ctx, "audit", "file", "file", "file_path=/vault/audit/audit.log")
	if err != nil {
		return err
	}

	err = client.EnsureCredentials(ctx)
	if err != nil {
		return err
	}

	namespaces := []string{"monadiam-apps", "monadiam-data", "monadiam-messaging", "monadiam-observability"}

	for _, namespace := range namespaces {
		err := client.put(ctx, namespace+"/bootstrap", map[string]string{"status": "configured"}, "")
		if err != nil {
			return err
		}
	}

	return nil
}

func (client Client) EnsureCredentials(ctx context.Context) error {
	for _, value := range credentialDefinitions(client.Environment) {
		err := client.ensureCredential(ctx, value)
		if err != nil {
			return err
		}
	}

	return client.ensureSeaweedFS(ctx)
}

func (client Client) execute(ctx context.Context, token string, input []byte, args ...string) (command.Result, error) {
	env := client.Environment
	argv := []string{
		"exec", "-i", "-n", env["VAULT_NAMESPACE"], env["VAULT_POD"], "--", "env",
		"VAULT_ADDR=" + env["VAULT_ADDR"],
		"VAULT_CACERT=" + env["VAULT_CACERT"],
	}

	if token != "" {
		argv = append(argv, "sh", "-c", tokenCommand, "sh")
		input = append([]byte(token+"\n"), input...)
	}

	argv = append(argv, "vault")
	argv = append(argv, args...)

	return client.Kubernetes.Execute(ctx, command.Request{
		Args:      argv,
		Input:     input,
		Capture:   true,
		Sensitive: true,
	})
}

func (client Client) readMaterial() (material, error) {
	data, err := os.ReadFile(client.Environment["VAULT_INIT_FILE"])
	if err != nil {
		return material{}, errors.New("cannot read Vault init file; restore matching init material")
	}

	var keys material
	err = json.Unmarshal(data, &keys)
	if err != nil || keys.RootToken == "" || strings.ContainsAny(keys.RootToken, "\r\n") || len(keys.Keys) == 0 {
		return material{}, errors.New("invalid Vault init material")
	}

	return keys, nil
}

func (client Client) authenticated(ctx context.Context, input []byte, args ...string) (command.Result, error) {
	keys, err := client.readMaterial()
	if err == nil {
		return client.execute(ctx, keys.RootToken, input, args...)
	} else {
		return command.Result{}, err
	}
}

func (client Client) put(ctx context.Context, path string, data map[string]string, cas string) error {
	input, err := json.Marshal(data)
	if err != nil {
		return err
	}

	args := []string{"kv", "put"}
	if cas != "" {
		args = append(args, "-cas="+cas)
	}

	args = append(args, client.Environment["VAULT_KV_MOUNT"]+"/"+path, "-")

	_, err = client.authenticated(ctx, input, args...)
	if err != nil {
		return fmt.Errorf("vault.Client.put: writing Vault secret %s: %w", path, err)
	}

	return nil
}

func (client Client) ensureMount(ctx context.Context, kind, name string, enable ...string) error {
	result, err := client.authenticated(ctx, nil, kind, "list", "-format=json")
	if err != nil {
		return err
	}

	var mounts map[string]json.RawMessage
	err = json.Unmarshal(result.Output, &mounts)
	if err != nil {
		return fmt.Errorf("vault.Client.ensureMount: invalid Vault %s list", kind)
	}

	_, exists := mounts[name+"/"]
	if exists {
		return nil
	}

	_, err = client.authenticated(ctx, nil, append([]string{kind, "enable"}, enable...)...)

	return err
}

func (client Client) configureRole(ctx context.Context, namespace, policy, role string) error {
	data, err := os.ReadFile("platform/vault/bootstrap/" + namespace + "-read.hcl")
	if err != nil {
		return err
	}

	_, err = client.authenticated(ctx, data, "policy", "write", policy, "-")
	if err != nil {
		return err
	}

	env := client.Environment
	_, err = client.authenticated(ctx, nil, "write", "auth/"+env["VAULT_K8S_AUTH_MOUNT"]+"/role/"+role,
		"bound_service_account_names=vault-auth", "bound_service_account_namespaces="+namespace,
		"policies="+policy, "audience="+env["VAULT_K8S_AUDIENCE"], "ttl="+env["VAULT_K8S_TTL"])

	return err
}

func (client Client) get(ctx context.Context, path string) (map[string]string, int, error) {
	result, err := client.authenticated(ctx, nil, "kv", "get", "-format=json", client.Environment["VAULT_KV_MOUNT"]+"/"+path)
	if err != nil {
		if strings.HasPrefix(strings.TrimSpace(string(result.Stderr)), "No value found at ") ||
			strings.HasPrefix(strings.TrimSpace(string(result.Output)), "No value found at ") {
			return nil, 0, nil
		}

		return nil, 0, fmt.Errorf("vault.Client.get: reading Vault secret %s: %w", path, err)
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

func (client Client) ensureSeaweedFS(ctx context.Context) error {
	const path = "monadiam-data/seaweedfs/main/auth"

	data, version, err := client.get(ctx, path)
	if err != nil {
		return err
	}

	keys := []string{
		"SEAWEEDFS_ADMIN_ACCESS_KEY",
		"SEAWEEDFS_ADMIN_SECRET_KEY",
		"SEAWEEDFS_ARCHIVER_ACCESS_KEY",
		"SEAWEEDFS_ARCHIVER_SECRET_KEY",
	}

	if data == nil {
		data = make(map[string]string)
		for _, key := range keys {
			random := make([]byte, 32)
			_, err := rand.Read(random)
			if err != nil {
				return err
			}

			data[key] = hex.EncodeToString(random)
		}
	}

	environment := maps.Clone(client.Environment)
	for _, key := range keys {
		environment[key] = data[key]
	}

	settings, err := seaweedfs.Load(environment)
	if err != nil {
		return err
	}

	document, err := seaweedfs.New(settings).Credentials()
	if err != nil {
		return err
	}

	if data["seaweedfs_s3_config"] == string(document) {
		return nil
	}

	data["seaweedfs_s3_config"] = string(document)

	return client.put(ctx, path, data, fmt.Sprint(version))
}

func backupMaterial(path string) error {
	_, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	} else if err != nil {
		return err
	}

	err = os.Chmod(path, 0600)
	if err == nil {
		return os.Rename(path, path+"."+time.Now().UTC().Format("20060102T150405.000000000Z")+".bak")
	} else {
		return err
	}
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
