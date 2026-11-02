package stand

import (
	"context"
	"time"

	"github.com/MonadIAM/infrastructure/cli/platform/command"
)

func (platform Platform) CertManager(ctx context.Context) error {
	_, err := platform.Runner.Run(ctx, command.Request{Name: "make", Args: []string{"ROOT=bootstrap", "terraform-apply"}})
	return err
}

func (platform Platform) MetricsServer(ctx context.Context) error {
	err := platform.Kubernetes.Run(ctx, "apply", "--server-side", "--field-manager=resource-budget", "-k", "clusters/"+platform.Environment["PROFILE"]+"/system")
	if err != nil {
		return err
	}

	err = platform.Kubernetes.Rollout(ctx, "kube-system", "deployment/metrics-server", "180s")
	if err != nil {
		return err
	}

	err = platform.Kubernetes.Wait(ctx, "", "apiservice/v1beta1.metrics.k8s.io", "condition=Available", "180s")
	if err != nil {
		return err
	}

	err = command.Poll(ctx, 180*time.Second, 5*time.Second, platform.metricsAvailable)
	if err != nil {
		return err
	}

	err = platform.Kubernetes.Run(ctx, "top", "nodes")
	if err != nil {
		return err
	}

	return platform.Kubernetes.Run(ctx, "top", "pods", "-A")
}

func (platform Platform) Istio(ctx context.Context) error {
	err := platform.installGatewayAPI(ctx)
	if err != nil {
		return err
	}

	err = platform.Helm.Repo(ctx, "istio", platform.Environment["ISTIO_HELM_REPO"])
	if err != nil {
		return err
	}

	for _, component := range istioComponents {
		err := platform.install(ctx, component.Release)
		if err != nil {
			return err
		}

		if component.Workload != "" {
			err = platform.Kubernetes.Rollout(ctx, "istio-system", component.Workload, "300s")
			if err != nil {
				return err
			}
		}
	}

	return platform.Kubernetes.ApplyKustomize(ctx, "infrastructure/networking/istio/resources")
}

func (platform Platform) Gateway(ctx context.Context) error {
	err := platform.Kubernetes.Rollout(ctx, "istio-system", "deployment/istiod", "180s")
	if err != nil {
		return err
	}

	data, err := platform.Helm.Execute(ctx, true, "template", "gateway", "infrastructure/networking/gateway-api",
		"--set-string", "applicationsIP="+platform.Environment["APPLICATIONS_IP"])
	if err != nil {
		return err
	}

	err = platform.Kubernetes.Apply(ctx, data, false)
	if err != nil {
		return err
	}

	err = platform.Kubernetes.Wait(ctx, "istio-ingress", "certificate/monadiam-localhost-gateway-tls", "condition=Ready", "180s")
	if err != nil {
		return err
	}

	err = platform.Kubernetes.Wait(ctx, "istio-ingress", "certificate/applications-gateway-tls", "condition=Ready", "180s")
	if err != nil {
		return err
	}

	conditions := []string{"Accepted", "Programmed"}

	for _, condition := range conditions {
		err := platform.Kubernetes.Wait(ctx, "istio-ingress", "gateway/public", "condition="+condition, "180s")
		if err != nil {
			return err
		}
	}

	return nil
}

func (platform Platform) metricsAvailable(ctx context.Context) (bool, error) {
	_, err := platform.Kubernetes.Capture(ctx, "top", "nodes")
	if err != nil {
		return false, nil
	}

	_, err = platform.Kubernetes.Capture(ctx, "top", "pods", "-A")

	return err == nil, nil
}

func (platform Platform) installGatewayAPI(ctx context.Context) error {
	err := platform.Kubernetes.ApplyKustomize(ctx, "clusters/local")
	if err != nil {
		return err
	}

	url := "https://github.com/kubernetes-sigs/gateway-api/releases/download/" + platform.Environment["GATEWAY_API_VERSION"] + "/standard-install.yaml"
	err = platform.Kubernetes.Run(ctx, "apply", "--server-side=true", "-f", url)
	if err != nil {
		return err
	}

	resources := []string{"gatewayclasses", "gateways", "httproutes", "referencegrants"}

	for _, resource := range resources {
		crd := "crd/" + resource + ".gateway.networking.k8s.io"
		err := platform.Kubernetes.Wait(ctx, "", crd, "condition=Established", "180s")
		if err != nil {
			return err
		}
	}

	return nil
}
