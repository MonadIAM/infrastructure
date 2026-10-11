package vault

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/MonadIAM/infrastructure/cli/platform/command"
)

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

func (client Client) Status(ctx context.Context) (Status, error) {
	err := client.Environment.Require("KUBE_CONTEXT", "VAULT_NAMESPACE", "VAULT_POD", "VAULT_ADDR", "VAULT_CACERT")
	if err != nil {
		return Status{}, err
	}

	result, err := client.execute(ctx, "", nil, "status", "-format=json")
	if err != nil && result.ExitCode != 2 {
		return Status{}, fmt.Errorf("vault status: %w", err)
	}

	var status Status
	err = json.Unmarshal(result.Output, &status)
	if err != nil || !bytes.Contains(result.Output, []byte(`"sealed"`)) {
		return Status{}, errors.New("invalid Vault status response")
	}

	return status, nil
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
		return fmt.Errorf("saving Vault init material: %w", writeErr)
	}

	syncErr := file.Sync()
	if syncErr != nil {
		return fmt.Errorf("syncing Vault init material: %w", syncErr)
	}

	if err != nil {
		return fmt.Errorf("vault initialization failed; inspect the protected init file: %w", err)
	}

	_, err = client.readMaterial()
	if err != nil {
		return err
	}

	_, err = fmt.Fprintln(client.Output, "Vault init material saved to", path)

	return err
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
			return fmt.Errorf("vault unseal: %w", err)
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
		return fmt.Errorf("writing Vault secret %s: %w", path, err)
	}

	return nil
}
