package stand

import (
	"context"

	"github.com/MonadIAM/infrastructure/cli/platform/command"
)

func (platform Platform) Operators(ctx context.Context) error {
	err := platform.Kubernetes.ApplyKustomize(ctx, "clusters/"+platform.Environment["PROFILE"]+"/infrastructure/namespaces")
	if err != nil {
		return err
	}

	err = platform.MonitoringCRDs(ctx)
	if err != nil {
		return err
	}

	_, err = platform.Runner.Run(ctx, command.Request{Name: "just", Args: []string{"tf::apply", "operators"}})
	if err != nil {
		return err
	}

	for _, operator := range operatorDefinitions() {
		err := platform.installOperator(ctx, operator)
		if err != nil {
			return err
		}
	}

	return nil
}

func (platform Platform) installOperator(ctx context.Context, operator operatorDefinition) error {
	deployment := "deployment/" + operator.Deployment
	if operator.Restart {
		err := platform.Kubernetes.Run(ctx, "rollout", "restart", deployment, "-n", operator.Namespace)
		if err != nil {
			return err
		}
	}

	err := platform.Kubernetes.Rollout(ctx, operator.Namespace, deployment, "600s")
	if err != nil {
		return err
	}

	for _, crd := range operator.CRDs {
		err := platform.Kubernetes.Wait(ctx, "", "crd/"+crd, "condition=Established", "600s")
		if err != nil {
			return err
		}
	}

	return nil
}
