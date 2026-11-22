package stand

import (
	"context"
)

func (platform Platform) MonitoringCRDs(ctx context.Context) error {
	err := platform.Helm.Repo(ctx, "prometheus-community", platform.Environment["PROMETHEUS_COMMUNITY_HELM_REPO"])
	if err != nil {
		return err
	}

	data, err := platform.Helm.Execute(ctx, true, "show", "crds", "prometheus-community/kube-prometheus-stack", "--version", platform.Environment["KUBE_PROMETHEUS_STACK_CHART_VERSION"])
	if err != nil {
		return err
	}

	err = platform.Kubernetes.Apply(ctx, data, true)
	if err != nil {
		return err
	}

	resources := []string{"podmonitors", "servicemonitors", "scrapeconfigs", "prometheuses", "alertmanagers", "prometheusrules"}

	for _, resource := range resources {
		crd := "crd/" + resource + ".monitoring.coreos.com"
		err := platform.Kubernetes.Wait(ctx, "", crd, "condition=Established", "180s")
		if err != nil {
			return err
		}
	}

	return nil
}
