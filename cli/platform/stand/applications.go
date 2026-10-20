package stand

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/MonadIAM/infrastructure/cli/platform/command"
)

func (platform Platform) Application(ctx context.Context, name, imageKey string) error {
	client := platform.VaultClient()
	err := client.Application(ctx, name)
	if err != nil {
		return err
	}

	err = platform.Kubernetes.ApplyKustomize(ctx, "applications/"+name+"/kafka")
	if err != nil {
		return err
	}

	err = platform.Kubernetes.Wait(ctx, messagingNamespace, "kafkauser.kafka.strimzi.io/"+name+"-service", "condition=Ready", "300s")
	if err != nil {
		return err
	}

	err = platform.Kubernetes.Wait(ctx, messagingNamespace, "secret/"+name+"-service", "create", "180s")
	if err != nil {
		return err
	}

	checksum, err := platform.syncApplicationSecrets(ctx, name)
	if err != nil {
		return err
	}

	data, err := platform.Kubernetes.Capture(ctx, "kustomize", "applications/"+name+"/resources")
	if err != nil {
		return err
	}

	manifest := strings.NewReplacer(
		"${APPLICATIONS_IP}", platform.Environment["APPLICATIONS_IP"],
		"${"+imageKey+"}", platform.Environment[imageKey],
		"${"+strings.TrimSuffix(imageKey, "_IMAGE")+"_CREDENTIALS_CHECKSUM}", checksum,
	).Replace(string(data))

	err = platform.Kubernetes.Apply(ctx, []byte(manifest), false)
	if err != nil {
		return err
	}

	for _, secret := range []string{name + "-postgresql", name + "-redis"} {
		err = platform.WaitSecret(ctx, appsNamespace, secret, secret)
		if err != nil {
			return err
		}
	}

	err = platform.Kubernetes.Wait(ctx, appsNamespace, "certificate/"+name+"-redis-client", "condition=Ready", "180s")
	if err != nil {
		return err
	}

	err = platform.Kubernetes.Rollout(ctx, appsNamespace, "deployment/"+name+"-service", "600s")
	if err != nil {
		diagnostics, cancel := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)
		defer cancel()

		_ = platform.Kubernetes.Run(diagnostics, "describe", "pods", "-n", appsNamespace, "-l", "app.kubernetes.io/name="+name+"-service")
		_ = platform.Kubernetes.Run(diagnostics, "logs", "deployment/"+name+"-service", "-n", appsNamespace, "--all-containers=true", "--tail=80", "--pod-running-timeout=5s")
	}

	return err
}

func (platform Platform) syncApplicationSecrets(ctx context.Context, name string) (string, error) {
	definitions := []struct {
		Namespace   string
		Source      string
		Destination string
		Keys        []string
	}{
		{Namespace: dataNamespace, Source: name + "-postgresql-ca", Destination: name + "-postgresql-ca", Keys: []string{"ca.crt"}},
		{Namespace: messagingNamespace, Source: "kafka-main-cluster-ca-cert", Destination: name + "-kafka-ca", Keys: []string{"ca.crt"}},
		{Namespace: messagingNamespace, Source: name + "-service", Destination: name + "-kafka-auth", Keys: []string{"password"}},
		{Namespace: "vault", Source: "vault-server-tls", Destination: name + "-vault-ca", Keys: []string{"ca.crt"}},
	}

	checksum := sha256.New()

	for _, definition := range definitions {
		result, err := platform.Kubernetes.Execute(ctx, command.Request{
			Args:      []string{"get", "secret", definition.Source, "-n", definition.Namespace, "-o", "json"},
			Capture:   true,
			Sensitive: true,
		})
		if err != nil {
			return "", err
		}

		var source applicationSecret
		err = json.Unmarshal(result.Output, &source)
		if err != nil {
			return "", fmt.Errorf("stand.Platform.syncApplicationSecrets: invalid secret %s/%s", definition.Namespace, definition.Source)
		}

		destination := applicationSecret{APIVersion: "v1", Kind: "Secret", Data: make(map[string]string)}
		destination.Metadata.Name = definition.Destination
		destination.Metadata.Namespace = appsNamespace

		for _, key := range definition.Keys {
			value := source.Data[key]
			if value == "" {
				return "", fmt.Errorf("stand.Platform.syncApplicationSecrets: missing key %s in %s/%s", key, definition.Namespace, definition.Source)
			}

			destination.Data[key] = value
		}

		data, err := json.Marshal(destination)
		if err != nil {
			return "", err
		}

		_, _ = checksum.Write(data)
		_, err = platform.Kubernetes.Execute(ctx, command.Request{
			Args:      []string{"apply", "--server-side", "--field-manager=stand-applications", "-f", "-"},
			Input:     data,
			Sensitive: true,
		})
		if err != nil {
			return "", err
		}
	}

	return hex.EncodeToString(checksum.Sum(nil)), nil
}
