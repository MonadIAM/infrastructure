package stand

import (
	"context"
)

func (platform Platform) Operators(ctx context.Context) error {
	err := platform.Kubernetes.ApplyKustomize(ctx, "clusters/local")
	if err != nil {
		return err
	}

	err = platform.MonitoringCRDs(ctx)
	if err != nil {
		return err
	}

	err = platform.Helm.Repo(ctx, "cnpg", platform.Environment["CNPG_HELM_REPO"])
	if err != nil {
		return err
	}

	err = platform.Helm.Repo(ctx, "altinity", platform.Environment["ALTINITY_HELM_REPO"])
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
	err := platform.install(ctx, operator.Release)
	if err != nil {
		return err
	}

	deployment := "deployment/" + operator.Deployment
	if operator.Restart {
		err := platform.Kubernetes.Run(ctx, "rollout", "restart", deployment, "-n", operator.Namespace)
		if err != nil {
			return err
		}
	}

	err = platform.Kubernetes.Rollout(ctx, operator.Namespace, deployment, "600s")
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
