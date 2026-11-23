package stand

import (
	"context"
	"time"

	"github.com/MonadIAM/infrastructure/cli/platform/command"
)

func (platform Platform) CertManager(ctx context.Context) error {
	_, err := platform.Runner.Run(ctx, command.Request{Name: "just", Args: []string{"tf::apply", "bootstrap"}})
	return err
}

func (platform Platform) MetricsServer(ctx context.Context) error {
	err := platform.Kubernetes.Rollout(ctx, "kube-system", "deployment/metrics-server", "180s")
	if err != nil {
		return err
	}

	err = platform.Kubernetes.Wait(ctx, "", "apiservice/v1beta1.metrics.k8s.io", "condition=Available", "180s")
	if err != nil {
		return err
	}

	return command.Poll(ctx, 180*time.Second, 5*time.Second, platform.metricsAvailable)
}

func (platform Platform) Istio(ctx context.Context) error {
	err := platform.installGatewayAPI(ctx)
	if err != nil {
		return err
	}

	_, err = platform.Runner.Run(ctx, command.Request{Name: "just", Args: []string{"tf::apply", "istio"}})
	if err != nil {
		return err
	}

	for _, workload := range []string{"deployment/istiod", "daemonset/istio-cni-node", "daemonset/ztunnel"} {
		err = platform.Kubernetes.Rollout(ctx, "istio-system", workload, "300s")
		if err != nil {
			return err
		}
	}

	return platform.Kubernetes.ApplyKustomize(ctx, "clusters/"+platform.Environment["PROFILE"]+"/infrastructure/istio/base")
}

func (platform Platform) Gateway(ctx context.Context) error {
	err := platform.Kubernetes.Rollout(ctx, "istio-system", "deployment/istiod", "180s")
	if err != nil {
		return err
	}

	_, err = platform.Runner.Run(ctx, command.Request{Name: "just", Args: []string{"tf::apply", "gateway"}})
	if err != nil {
		return err
	}

	err = platform.Kubernetes.Wait(ctx, "istio-ingress", "deployment/public-istio", "condition=Available", "180s")
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
	err := platform.Kubernetes.ApplyKustomize(ctx, "clusters/"+platform.Environment["PROFILE"]+"/infrastructure/namespaces")
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
