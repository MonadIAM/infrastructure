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

func (platform Platform) Observability(ctx context.Context) error {
	err := platform.Kubernetes.ApplyKustomize(ctx, "clusters/local")
	if err != nil {
		return err
	}

	err = platform.Kubernetes.Wait(ctx, observabilityNamespace, "secret/grafana-admin", "create", "180s")
	if err != nil {
		return err
	}

	err = platform.MonitoringCRDs(ctx)
	if err != nil {
		return err
	}

	err = platform.installPrometheus(ctx)
	if err != nil {
		return err
	}

	err = platform.installTelemetry(ctx)
	if err != nil {
		return err
	}

	return platform.Kubernetes.Run(ctx, "get", "pods,svc", "-n", observabilityNamespace)
}

func (platform Platform) installPrometheus(ctx context.Context) error {
	err := platform.install(ctx, "kube-prometheus-stack")
	if err != nil {
		return err
	}

	workloads := []string{
		"deployment/kube-prometheus-stack-operator",
		"deployment/kube-prometheus-stack-grafana",
		"deployment/kube-prometheus-stack-kube-state-metrics",
		"daemonset/kube-prometheus-stack-prometheus-node-exporter",
	}

	for _, workload := range workloads {
		err := platform.Kubernetes.Rollout(ctx, observabilityNamespace, workload, "300s")
		if err != nil {
			return err
		}
	}

	resources := []string{"prometheus/kube-prometheus-stack-prometheus", "alertmanager/kube-prometheus-stack-alertmanager"}

	for _, resource := range resources {
		err := platform.Kubernetes.Wait(ctx, observabilityNamespace, resource, "condition=Available", "300s")
		if err != nil {
			return err
		}
	}

	return nil
}

func (platform Platform) installTelemetry(ctx context.Context) error {
	err := platform.Helm.Repo(ctx, "grafana-community", platform.Environment["GRAFANA_COMMUNITY_HELM_REPO"])
	if err != nil {
		return err
	}

	err = platform.Helm.Repo(ctx, "open-telemetry", platform.Environment["OPENTELEMETRY_HELM_REPO"])
	if err != nil {
		return err
	}

	releases := []string{"loki", "tempo", "otel-collector"}

	for _, name := range releases {
		err := platform.install(ctx, name)
		if err != nil {
			return err
		}

		workload := "statefulset/" + name
		if name == "otel-collector" {
			workload = "deployment/" + name
		}

		err = platform.Kubernetes.Rollout(ctx, observabilityNamespace, workload, "300s")
		if err != nil {
			return err
		}
	}

	return nil
}
