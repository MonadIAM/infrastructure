package stand

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/MonadIAM/infrastructure/cli/platform/command"
)

func (platform Platform) copyKafkaArchiveSecrets(ctx context.Context) error {
	for _, name := range []string{"kafka-main-archive", "kafka-main-cluster-ca-cert"} {
		key := "password"
		if name == "kafka-main-cluster-ca-cert" {
			key = "ca.crt"
		}

		err := platform.Kubernetes.Wait(ctx, messagingNamespace, "secret/"+name, "create", "300s")
		if err != nil {
			return err
		}

		var source struct {
			Data map[string]string `json:"data"`
			Type string            `json:"type"`
		}
		err = platform.Kubernetes.Get(ctx, messagingNamespace, "secret/"+name, &source)
		if err != nil {
			return err
		}

		if source.Data[key] == "" {
			return errors.New("kafka secret is missing required data")
		}

		manifest, err := json.Marshal(map[string]any{
			"apiVersion": "v1",
			"kind":       "Secret",
			"metadata": map[string]string{
				"name":      name,
				"namespace": dataNamespace,
			},
			"type": source.Type,
			"data": map[string]string{key: source.Data[key]},
		})
		if err != nil {
			return err
		}

		_, err = platform.Kubernetes.Execute(ctx, command.Request{
			Args:      []string{"apply", "-f", "-"},
			Input:     manifest,
			Sensitive: true,
		})
		if err != nil {
			return fmt.Errorf("copy Kafka secret %s to %s: %w", name, dataNamespace, err)
		}
	}

	return nil
}
