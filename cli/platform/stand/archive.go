package stand

import "context"

func (platform Platform) configureArchive(ctx context.Context, image string) error {
	data, err := platform.Helm.Execute(ctx, true, "template", "clickhouse-archive",
		"clusters/"+platform.Environment["PROFILE"]+"/infrastructure/clickhouse/archive/chart",
		"--namespace", dataNamespace,
		"--values", "clusters/"+platform.Environment["PROFILE"]+"/infrastructure/clickhouse/archive-values.yaml",
		"--set-string", "image="+image)
	if err != nil {
		return err
	}

	err = platform.Kubernetes.Run(ctx, "delete", "job", "clickhouse-archive-bootstrap", "-n", dataNamespace,
		"--ignore-not-found", "--wait=true", "--timeout=60s")
	if err != nil {
		return err
	}

	err = platform.Kubernetes.Apply(ctx, data, false)
	if err != nil {
		return err
	}

	err = platform.Kubernetes.Wait(ctx, dataNamespace, "job/clickhouse-archive-bootstrap", "condition=Complete", "600s")
	if err != nil {
		_ = platform.Kubernetes.Run(ctx, "logs", "job/clickhouse-archive-bootstrap", "-n", dataNamespace,
			"--all-containers=true", "--tail=50")
		return err
	}

	return nil
}
