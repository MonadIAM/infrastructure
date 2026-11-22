package stand

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/MonadIAM/infrastructure/cli/platform/command"
)

func (platform Platform) Vault(ctx context.Context) error {
	err := platform.installVault(ctx)
	if err != nil {
		return err
	}

	client := platform.VaultClient()
	err = client.Initialize(ctx)
	if err != nil {
		return err
	}

	err = client.Unseal(ctx)
	if err != nil {
		return err
	}

	err = client.Bootstrap(ctx)
	if err != nil {
		return err
	}

	err = platform.installVaultOperator(ctx)
	if err != nil {
		return err
	}

	return platform.waitVaultSecrets(ctx)
}

func (platform Platform) installVault(ctx context.Context) error {
	_, err := platform.Runner.Run(ctx, command.Request{Name: "make", Args: []string{"ROOT=vault-server", "terraform-apply"}})
	if err != nil {
		return err
	}

	for _, condition := range []string{"create", "condition=Ready"} {
		err := platform.Kubernetes.Wait(ctx, platform.Environment["VAULT_NAMESPACE"], "pod/"+platform.Environment["VAULT_POD"], condition, "300s")
		if err != nil {
			return err
		}
	}

	return nil
}

func (platform Platform) installVaultOperator(ctx context.Context) error {
	_, err := platform.Runner.Run(ctx, command.Request{Name: "make", Args: []string{"ROOT=vault-operator", "terraform-apply"}})
	if err != nil {
		return err
	}

	err = platform.Kubernetes.Run(ctx, "rollout", "restart", vaultOperatorDeployment, "-n", vaultOperatorNamespace)
	if err != nil {
		return err
	}

	err = platform.Kubernetes.Rollout(ctx, vaultOperatorNamespace, vaultOperatorDeployment, "180s")
	if err != nil {
		return err
	}

	return platform.Kubernetes.ApplyKustomize(ctx, "infrastructure/vault-secrets-operator/base")
}

func (platform Platform) waitVaultSecrets(ctx context.Context) error {
	err := platform.waitVaultPrerequisites(ctx)
	if err != nil {
		return err
	}

	data, err := platform.Kubernetes.Capture(ctx, "get", "-k", "infrastructure/vault-secrets-operator/base", "-o", "json")
	if err != nil {
		return fmt.Errorf("stand.Platform.waitVaultSecrets: get Kustomize resources: %w", err)
	}

	var objects struct {
		Items []staticSecret `json:"items"`
	}

	err = json.Unmarshal(data, &objects)
	if err != nil {
		return fmt.Errorf("stand.Platform.waitVaultSecrets: decode Kustomize resources: %w", err)
	}

	count := 0
	for _, secret := range objects.Items {
		if secret.Kind != "VaultStaticSecret" {
			continue
		}

		if secret.Metadata.Name == "" || secret.Metadata.Namespace == "" || secret.Spec.Destination.Name == "" {
			return fmt.Errorf("stand.Platform.waitVaultSecrets: incomplete VaultStaticSecret")
		}

		count++
		err = platform.WaitSecret(ctx, secret.Metadata.Namespace, secret.Metadata.Name, secret.Spec.Destination.Name)
		if err != nil {
			return err
		}
	}

	if count == 0 {
		return fmt.Errorf("stand.Platform.waitVaultSecrets: no VaultStaticSecret resources")
	}

	return nil
}

func (platform Platform) waitVaultPrerequisites(ctx context.Context) error {
	resources := []string{"certificate/vault-server-ca", "vaultconnections.secrets.hashicorp.com/vault"}

	for _, resource := range resources {
		err := platform.Kubernetes.Wait(ctx, vaultOperatorNamespace, resource, "condition=Ready", "180s")
		if err != nil {
			return err
		}
	}

	namespaces := []string{"monadiam-apps", dataNamespace, messagingNamespace, observabilityNamespace, "clickhouse-operator"}

	for _, namespace := range namespaces {
		resource := "vaultauths.secrets.hashicorp.com/" + namespace
		err := platform.Kubernetes.Wait(ctx, namespace, resource, "condition=Ready", "180s")
		if err != nil {
			return err
		}
	}

	return nil
}
