package stand

import (
	"context"
	"fmt"

	"github.com/MonadIAM/infrastructure/cli/platform/command"
)

func (platform Platform) Status(ctx context.Context) error {
	err := platform.Environment.Require("KUBE_CONTEXT")
	if err != nil {
		return err
	}

	err = platform.Kubernetes.Run(ctx, "get", "nodes", "-o", "wide")
	if err != nil {
		return err
	}

	err = platform.Kubernetes.Run(ctx, "get", "namespaces")
	if err != nil {
		return err
	}

	_, err = platform.Runner.Run(ctx, command.Request{
		Name: "helm",
		Args: []string{"--kube-context", platform.Environment["KUBE_CONTEXT"], "list", "-A"},
	})
	if err != nil {
		return err
	}

	commands := [][]string{
		{"get", "cluster.postgresql.cnpg.io", "-n", "monadiam-data"},
		{"get", "pods,svc", "-l", "app.kubernetes.io/name=redis", "-n", "monadiam-data"},
		{"get", kafkaResources, "-n", "monadiam-messaging"},
		{"get", "deployment,pods,svc", "-n", messagingNamespace, "-l", "app.kubernetes.io/name=schema-registry-main"},
		{"get", "jobs,pods", "-n", messagingNamespace, "-l", "app.kubernetes.io/name=shared-schemas"},
		{"get", "clickhouseinstallations.clickhouse.altinity.com", "-n", "monadiam-data"},
		{"get", "pods,svc", "-n", "monadiam-observability"},
		{"get", "deployment,pods,svc", "-n", appsNamespace},
	}

	for _, args := range commands {
		err := ctx.Err()
		if err != nil {
			return err
		}

		err = platform.Kubernetes.Run(ctx, args...)
		if err != nil {
			return fmt.Errorf("stand.Platform.Status: %s: %w", args[1], err)
		}
	}

	return nil
}
