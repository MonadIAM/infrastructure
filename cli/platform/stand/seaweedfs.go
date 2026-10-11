package stand

import (
	"context"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"

	"github.com/MonadIAM/infrastructure/cli/platform/command"
)

func (platform Platform) SeaweedFS(ctx context.Context) error {
	err := platform.MonitoringCRDs(ctx)
	if err != nil {
		return err
	}

	err = platform.Kubernetes.ApplyKustomize(ctx, "platform/seaweedfs/resources")
	if err != nil {
		return err
	}

	secrets := []string{"seaweedfs-main-s3", "seaweedfs-main-admin", "seaweedfs-main-archiver"}
	for _, secret := range secrets {
		err := platform.WaitSecret(ctx, dataNamespace, secret, secret)
		if err != nil {
			return err
		}
	}

	err = platform.Helm.Repo(ctx, "seaweedfs", platform.Environment["SEAWEEDFS_HELM_REPO"])
	if err != nil {
		return err
	}

	err = platform.install(ctx, "seaweedfs-main")
	if err != nil {
		return err
	}

	err = platform.Kubernetes.ApplyKustomize(ctx, "platform/seaweedfs/monitoring")
	if err != nil {
		return err
	}

	image, err := platform.seaweedFSImage(ctx)
	if err != nil {
		return err
	}

	return platform.configureSeaweedFS(ctx, image)
}

func (platform Platform) seaweedFSImage(ctx context.Context) (string, error) {
	const tag = "monadiam/stand-cli:seaweedfs"

	_, err := platform.Runner.Run(ctx, command.Request{
		Name: "docker",
		Args: []string{"build", "--file", "platform/seaweedfs/Dockerfile", "--tag", tag, "."},
	})
	if err != nil {
		return "", err
	}

	result, err := platform.Runner.Run(ctx, command.Request{
		Name:    "docker",
		Args:    []string{"image", "inspect", "--format", "{{.Id}}", tag},
		Capture: true,
	})
	if err != nil {
		return "", err
	}

	digest := strings.TrimPrefix(strings.TrimSpace(string(result.Output)), "sha256:")
	decoded, err := hex.DecodeString(digest)
	if err != nil || len(decoded) != 32 {
		return "", fmt.Errorf("invalid CLI image digest")
	}

	image := "monadiam/stand-cli:" + digest
	_, err = platform.Runner.Run(ctx, command.Request{
		Name: "docker",
		Args: []string{"image", "tag", tag, image},
	})
	if err != nil {
		return "", err
	}

	_, err = platform.Runner.Run(ctx, command.Request{
		Name: "k3d",
		Args: []string{"image", "import", image, "--cluster", platform.Environment["CLUSTER"]},
	})

	return image, err
}

func (platform Platform) configureSeaweedFS(ctx context.Context, image string) error {
	data, err := platform.Kubernetes.Capture(ctx, "kustomize", "platform/seaweedfs/bootstrap")
	if err != nil {
		return err
	}

	rendered, err := platform.seaweedFSManifest(string(data), image)
	if err != nil {
		return err
	}

	err = platform.Kubernetes.Run(ctx, "delete", "job", "seaweedfs-main-bootstrap", "-n", dataNamespace, "--ignore-not-found", "--wait=true", "--timeout=60s")
	if err != nil {
		return err
	}

	err = platform.Kubernetes.Apply(ctx, []byte(rendered), false)
	if err != nil {
		return err
	}

	err = platform.Kubernetes.Wait(ctx, dataNamespace, "job/seaweedfs-main-bootstrap", "condition=Complete", "300s")
	if err != nil {
		_ = platform.Kubernetes.Run(ctx, "logs", "job/seaweedfs-main-bootstrap", "-n", dataNamespace, "--all-containers=true", "--tail=50")
		return err
	}

	return nil
}

func (platform Platform) seaweedFSManifest(manifest, image string) (string, error) {
	keys := []string{
		"SEAWEEDFS_ENDPOINT",
		"SEAWEEDFS_REGION",
		"SEAWEEDFS_AUDIT_LOG_BUCKET",
		"SEAWEEDFS_AUDIT_LOG_RETENTION_DAYS",
		"SEAWEEDFS_CHANGE_LOG_BUCKET",
		"SEAWEEDFS_CHANGE_LOG_RETENTION_DAYS",
	}

	manifest = strings.ReplaceAll(manifest, "${SEAWEEDFS_BOOTSTRAP_IMAGE}", image)
	for _, key := range keys {
		value := platform.Environment[key]
		if value == "" || strings.ContainsAny(value, "\r\n\"'\\") {
			return "", fmt.Errorf("invalid %s for SeaweedFS bootstrap", key)
		}

		placeholder := "${" + key + "}"
		replacer := strings.NewReplacer(
			"\""+placeholder+"\"", strconv.Quote(value),
			"'"+placeholder+"'", strconv.Quote(value),
			placeholder, strconv.Quote(value),
		)
		manifest = replacer.Replace(manifest)
	}

	return manifest, nil
}
