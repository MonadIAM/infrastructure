package stand

import "context"

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
	err := platform.Kubernetes.ApplyKustomize(ctx, "clusters/local")
	if err != nil {
		return err
	}

	err = platform.Helm.Repo(ctx, "hashicorp", platform.Environment["HASHICORP_HELM_REPO"])
	if err != nil {
		return err
	}

	err = platform.Kubernetes.ApplyKustomize(ctx, "platform/vault/resources")
	if err != nil {
		return err
	}

	namespace := platform.Environment["VAULT_NAMESPACE"]
	err = platform.Kubernetes.Wait(ctx, namespace, "certificate/vault-server-tls", "condition=Ready", "180s")
	if err != nil {
		return err
	}

	err = platform.install(ctx, "vault")
	if err != nil {
		return err
	}

	pod := "pod/" + platform.Environment["VAULT_POD"]
	conditions := []string{"create", "condition=PodScheduled", "condition=Ready"}

	for _, condition := range conditions {
		err := platform.Kubernetes.Wait(ctx, namespace, pod, condition, "300s")
		if err != nil {
			return err
		}
	}

	return nil
}

func (platform Platform) installVaultOperator(ctx context.Context) error {
	err := platform.install(ctx, "vault-secrets-operator")
	if err != nil {
		return err
	}

	err = platform.Kubernetes.Rollout(ctx, vaultOperatorNamespace, vaultOperatorDeployment, "180s")
	if err != nil {
		return err
	}

	err = platform.Kubernetes.ApplyKustomize(ctx, "infrastructure/security/vault-secrets-operator/resources")
	if err != nil {
		return err
	}

	err = platform.Kubernetes.Run(ctx, "rollout", "restart", vaultOperatorDeployment, "-n", vaultOperatorNamespace)
	if err != nil {
		return err
	}

	return platform.Kubernetes.Rollout(ctx, vaultOperatorNamespace, vaultOperatorDeployment, "180s")
}

func (platform Platform) waitVaultSecrets(ctx context.Context) error {
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

	for _, secret := range secretDefinitions() {
		err := platform.WaitSecret(ctx, secret.Namespace, secret.Name, secret.Destination)
		if err != nil {
			return err
		}
	}

	return nil
}
