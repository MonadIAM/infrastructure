package stand

import "context"

var apicurioKafkaResources = []string{
	"kafkatopic.kafka.strimzi.io/kafkasql-journal",
	"kafkatopic.kafka.strimzi.io/kafkasql-snapshots",
	"kafkatopic.kafka.strimzi.io/registry-events",
	"kafkauser.kafka.strimzi.io/schema-registry-main",
}

func (platform Platform) Apicurio(ctx context.Context) error {
	err := platform.Kubernetes.Wait(ctx, messagingNamespace, "kafka.kafka.strimzi.io/kafka-main", "condition=Ready", "900s")
	if err != nil {
		return err
	}

	err = platform.Kubernetes.ApplyKustomize(ctx, "platform/apicurio/kafka")
	if err != nil {
		return err
	}

	for _, resource := range apicurioKafkaResources {
		err = platform.Kubernetes.Wait(ctx, messagingNamespace, resource, "condition=Ready", "300s")
		if err != nil {
			return err
		}
	}

	err = platform.Kubernetes.Wait(ctx, messagingNamespace, "secret/schema-registry-main", "create", "180s")
	if err != nil {
		return err
	}

	err = platform.Kubernetes.ApplyKustomize(ctx, "platform/apicurio/resources")
	if err != nil {
		return err
	}

	return platform.Kubernetes.Rollout(ctx, messagingNamespace, "deployment/schema-registry-main", "600s")
}
