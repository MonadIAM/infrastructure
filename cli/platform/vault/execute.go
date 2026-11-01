package vault

import (
	"context"
	"errors"
	"maps"
	"net/url"
	"os"
	"path/filepath"

	"github.com/MonadIAM/infrastructure/cli/platform/command"
)

func (client Client) Execute(ctx context.Context, arguments ...string) error {
	if len(arguments) == 0 {
		return errors.New("command is required after --")
	}

	material, err := client.readMaterial()
	if err != nil {
		return err
	}

	address, err := url.Parse(client.Environment["VAULT_ADDR"])
	if err != nil {
		return err
	}

	ca, err := client.Kubernetes.Capture(ctx, "exec", "-n", client.Environment["VAULT_NAMESPACE"], client.Environment["VAULT_POD"], "--", "cat", client.Environment["VAULT_CACERT"])
	if err != nil {
		return err
	}

	environment := maps.Clone(client.Environment)
	environment["VAULT_CACERT"] = filepath.Join(environment["VAULT_STATE_DIR"], "ca.crt")
	err = os.WriteFile(environment["VAULT_CACERT"], ca, 0600)
	if err != nil {
		return err
	}

	forwarded, closeForward, err := client.Kubernetes.Forward(ctx, environment["VAULT_NAMESPACE"], environment["VAULT_POD"], "8200", environment["KUBECONFIG"])
	if err != nil {
		return err
	}

	defer closeForward()
	environment["VAULT_ADDR"] = "https://" + forwarded
	environment["VAULT_TLS_SERVER_NAME"] = address.Hostname()
	environment["VAULT_TOKEN"] = material.RootToken
	executor := command.Executor{Environment: environment, Output: client.Output, Errors: client.Output}
	_, err = executor.Run(ctx, command.Request{Name: arguments[0], Args: arguments[1:]})
	return err
}
