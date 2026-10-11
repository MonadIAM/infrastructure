package stand

import (
	"context"
	"strings"
	"time"
)

func (platform Platform) KafkaResources(ctx context.Context) error {
	_, err := platform.Environment.PositiveInt("PROJECTION_JOURNAL_PARTITIONS")
	if err != nil {
		return err
	}

	data, err := platform.Kubernetes.Capture(ctx, "kustomize", "platform/kafka/resources")
	if err != nil {
		return err
	}

	rendered := strings.ReplaceAll(string(data), "${PROJECTION_JOURNAL_PARTITIONS}", platform.Environment["PROJECTION_JOURNAL_PARTITIONS"])

	return platform.Kubernetes.Apply(ctx, []byte(rendered), false)
}

func (platform Platform) Clusters(ctx context.Context) error {
	err := platform.applyClusterResources(ctx)
	if err != nil {
		return err
	}

	err = platform.installRedis(ctx)
	if err != nil {
		return err
	}

	err = platform.Kubernetes.Wait(ctx, dataNamespace, "cluster.postgresql.cnpg.io/postgresql-main", "condition=Ready", "600s")
	if err != nil {
		return err
	}

	err = platform.waitKafka(ctx)
	if err != nil {
		return err
	}

	return platform.Kubernetes.WaitState(ctx, dataNamespace, "clickhouseinstallations.clickhouse.altinity.com/clickhouse-main", "status", "Completed", 600*time.Second)
}

func (platform Platform) applyClusterResources(ctx context.Context) error {
	err := platform.MonitoringCRDs(ctx)
	if err != nil {
		return err
	}

	directories := []string{"postgresql", "redis", "clickhouse"}

	for _, directory := range directories {
		err := platform.Kubernetes.ApplyKustomize(ctx, "platform/"+directory+"/resources")
		if err != nil {
			return err
		}
	}

	return platform.KafkaResources(ctx)
}

func (platform Platform) installRedis(ctx context.Context) error {
	err := platform.Kubernetes.Wait(ctx, dataNamespace, "certificate/redis-main-tls", "condition=Ready", "180s")
	if err != nil {
		return err
	}

	err = platform.Kubernetes.Wait(ctx, dataNamespace, "secret/redis-main-auth", "create", "180s")
	if err != nil {
		return err
	}

	return platform.install(ctx, "redis-main")
}

func (platform Platform) waitKafka(ctx context.Context) error {
	err := platform.Kubernetes.WaitPods(ctx, messagingNamespace, "strimzi.io/name=kafka-main-kafka", "900s")
	if err != nil {
		return err
	}

	err = platform.Kubernetes.Wait(ctx, messagingNamespace, "kafka.kafka.strimzi.io/kafka-main", "condition=Ready", "900s")
	if err != nil {
		return err
	}

	err = platform.Kubernetes.Rollout(ctx, messagingNamespace, "deployment/kafka-main-entity-operator", "300s")
	if err != nil {
		return err
	}

	resources := []string{
		"kafkatopic.kafka.strimzi.io/connect-configs",
		"kafkatopic.kafka.strimzi.io/connect-offsets",
		"kafkatopic.kafka.strimzi.io/connect-status",
		"kafkauser.kafka.strimzi.io/kafka-main-admin",
		"kafkauser.kafka.strimzi.io/kafka-connect",
	}

	for _, resource := range resources {
		err := platform.Kubernetes.Wait(ctx, messagingNamespace, resource, "condition=Ready", "300s")
		if err != nil {
			return err
		}
	}

	err = platform.Kubernetes.WaitPods(ctx, messagingNamespace, "strimzi.io/name=kafka-connect-connect", "900s")
	if err != nil {
		return err
	}

	err = platform.Kubernetes.Wait(ctx, messagingNamespace, "kafkaconnect.kafka.strimzi.io/kafka-connect", "condition=Ready", "900s")
	if err != nil {
		return err
	}

	return platform.Kubernetes.Wait(ctx, messagingNamespace, "kafkaconnector.kafka.strimzi.io/postgresql-main-outbox", "condition=Ready", "300s")
}
