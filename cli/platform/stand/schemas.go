package stand

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/MonadIAM/infrastructure/cli/platform/command"
)

func (platform Platform) Schemas(ctx context.Context) error {
	err := platform.Kubernetes.Rollout(ctx, messagingNamespace, "deployment/schema-registry-main", "600s")
	if err != nil {
		return err
	}

	data, err := platform.Kubernetes.Capture(ctx, "kustomize", "platform/apicurio/schemas")
	if err != nil {
		return err
	}

	manifest := strings.ReplaceAll(string(data), "${SHARED_SCHEMAS_IMAGE}", platform.Environment["SHARED_SCHEMAS_IMAGE"])
	err = platform.Kubernetes.Run(ctx, "delete", "job", "shared-schemas", "-n", messagingNamespace, "--ignore-not-found", "--wait=true", "--timeout=60s")
	if err != nil {
		return err
	}

	err = platform.Kubernetes.Apply(ctx, []byte(manifest), false)
	if err != nil {
		return err
	}

	err = platform.waitSchemas(ctx)
	if err != nil {
		diagnostics, cancel := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)
		defer cancel()

		_ = platform.Kubernetes.Run(diagnostics, "describe", "job", "shared-schemas", "-n", messagingNamespace)
		_ = platform.Kubernetes.Run(diagnostics, "describe", "pods", "-n", messagingNamespace, "-l", "job-name=shared-schemas")
		_ = platform.Kubernetes.Run(diagnostics, "logs", "job/shared-schemas", "-n", messagingNamespace, "--all-containers=true", "--tail=100", "--pod-running-timeout=5s")

		return err
	}

	return platform.Kubernetes.Run(ctx, "logs", "job/shared-schemas", "-n", messagingNamespace, "--tail=100")
}

func (platform Platform) waitSchemas(ctx context.Context) error {
	err := command.Poll(ctx, 6*time.Minute, 2*time.Second, func(ctx context.Context) (bool, error) {
		var job schemaJob

		err := platform.Kubernetes.Get(ctx, messagingNamespace, "job/shared-schemas", &job)
		if err != nil {
			return false, err
		}

		for _, condition := range job.Status.Conditions {
			if condition.Status == "True" {
				switch condition.Type {
				case "Complete":
					return true, nil
				case "Failed", "FailureTarget":
					return false, fmt.Errorf("shared-schemas Job failed: %s: %s", condition.Reason, condition.Message)
				}
			}
		}

		return false, nil
	})
	if err != nil {
		return fmt.Errorf("stand.Platform.waitSchemas: %w", err)
	}

	return nil
}
