package stand

import (
	"context"
	"fmt"
	"time"

	"github.com/MonadIAM/infrastructure/cli/platform/command"
)

func (secret staticSecret) ready() bool {
	conditions := map[string]string{}
	for _, condition := range secret.Status.Conditions {
		conditions[condition.Type] = condition.Status
	}

	return secret.Metadata.Generation > 0 && secret.Metadata.Generation == secret.Status.LastGeneration &&
		secret.Status.SecretMAC != "" && conditions["Ready"] == "True" && conditions["Healthy"] == "True"
}

func (platform Platform) WaitSecret(ctx context.Context, namespace, name, destination string) error {
	err := command.Poll(ctx, 180*time.Second, 5*time.Second, func(ctx context.Context) (bool, error) {
		var secret staticSecret

		err := platform.Kubernetes.Get(ctx, namespace, "vaultstaticsecrets.secrets.hashicorp.com/"+name, &secret)
		if err == nil {
			return secret.ready(), nil
		} else {
			return false, err
		}
	})
	if err != nil {
		return fmt.Errorf("waiting for VaultStaticSecret %s/%s: %w", namespace, name, err)
	}

	return platform.Kubernetes.Wait(ctx, namespace, "secret/"+destination, "create", "180s")
}
