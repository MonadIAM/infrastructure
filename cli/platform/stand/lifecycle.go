package stand

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/MonadIAM/infrastructure/cli/platform/command"
)

func (platform Platform) ClusterExists(ctx context.Context) (bool, error) {
	result, err := platform.Runner.Run(ctx, command.Request{
		Name:    "k3d",
		Args:    []string{"cluster", "list", "-o", "json"},
		Capture: true,
	})
	if err != nil {
		return false, err
	}

	var clusters []clusterSummary
	err = json.Unmarshal(result.Output, &clusters)
	if err != nil {
		return false, errors.New("invalid k3d cluster list")
	}

	for _, cluster := range clusters {
		if cluster.Name == platform.Environment["CLUSTER"] {
			return true, nil
		}
	}

	return false, nil
}

func (platform Platform) Start(ctx context.Context, resume bool) error {
	err := platform.Environment.Validate()
	if err != nil {
		return err
	}

	exists, err := platform.ClusterExists(ctx)
	if err != nil {
		return err
	}

	if resume && !exists {
		return errors.New("cluster does not exist; use make start")
	}

	args := []string{"cluster", "create", platform.Environment["CLUSTER"], "--config", platform.Environment["K3D_CONFIG"], "--kubeconfig-switch-context=false"}
	if exists {
		args = []string{"cluster", "start", platform.Environment["CLUSTER"], "--timeout", "180s"}
	}

	_, err = platform.Runner.Run(ctx, command.Request{
		Name: "k3d",
		Args: args,
	})
	if err != nil {
		return err
	}

	err = platform.Kubernetes.Run(ctx, "wait", "--for=condition=Ready", "node", "--all", "--timeout=180s")
	if err != nil {
		return err
	}

	err = platform.Kubernetes.ApplyKustomize(ctx, "clusters/local")
	if err != nil {
		return err
	}

	err = platform.Reconcile(ctx, "all")
	if err != nil {
		return err
	}

	return platform.Status(ctx)
}

func (platform Platform) Stop(ctx context.Context) error {
	err := platform.Environment.Require("CLUSTER")
	if err != nil {
		return err
	}

	exists, err := platform.ClusterExists(ctx)
	if err != nil {
		return err
	}

	if !exists {
		_, err := fmt.Fprintln(platform.Output, "Cluster does not exist.")

		return err
	}

	_, err = platform.Runner.Run(ctx, command.Request{
		Name: "k3d",
		Args: []string{"cluster", "stop", platform.Environment["CLUSTER"]},
	})

	return err
}

func (platform Platform) Restart(ctx context.Context) error {
	err := platform.Environment.Validate()
	if err != nil {
		return err
	}

	err = platform.Stop(ctx)
	if err == nil {
		return platform.Start(ctx, false)
	} else {
		return err
	}
}

func (platform Platform) Recreate(ctx context.Context, confirmed bool) error {
	if !confirmed {
		return errors.New("cluster recreation deletes the cluster; pass --confirm-delete")
	}

	err := platform.Environment.Validate()
	if err != nil {
		return err
	}

	exists, err := platform.ClusterExists(ctx)
	if err != nil {
		return err
	}

	if exists {
		_, err = platform.Runner.Run(ctx, command.Request{
			Name: "k3d",
			Args: []string{"cluster", "delete", platform.Environment["CLUSTER"]},
		})
		if err != nil {
			return err
		}
	}

	return platform.Start(ctx, false)
}

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
			_, _ = fmt.Fprintln(platform.Output, "Component status unavailable:", args[1])
		}
	}

	return nil
}
