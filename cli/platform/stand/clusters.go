package stand

import (
	"context"
	"time"

	"github.com/MonadIAM/infrastructure/cli/platform/command"
)

func (platform Platform) KafkaResources(ctx context.Context) error {
	return platform.Kubernetes.ApplyKustomize(ctx, "clusters/"+platform.Environment["PROFILE"]+"/infrastructure/kafka")
}

func (platform Platform) Clusters(ctx context.Context) error {
	err := platform.applyClusterResources(ctx)
	if err != nil {
		return err
	}

	_, err = platform.Runner.Run(ctx, command.Request{Name: "just", Args: []string{"tf::apply", "redis"}})
	if err != nil {
		return err
	}

	err = platform.Kubernetes.Run(ctx, "wait", "--for=condition=Ready", "cluster.postgresql.cnpg.io", "-l", "app.kubernetes.io/part-of=monadiam,app.kubernetes.io/component=database", "-n", dataNamespace, "--timeout=600s")
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

	directories := []string{
		"clusters/" + platform.Environment["PROFILE"] + "/infrastructure/postgresql",
		"clusters/" + platform.Environment["PROFILE"] + "/infrastructure/redis/base",
		"clusters/" + platform.Environment["PROFILE"] + "/infrastructure/clickhouse",
	}

	for _, directory := range directories {
		err := platform.Kubernetes.ApplyKustomize(ctx, directory)
		if err != nil {
			return err
		}
	}

	return platform.KafkaResources(ctx)
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

	err = platform.Kubernetes.Run(ctx, "wait", "kafkatopic.kafka.strimzi.io", "-n", messagingNamespace, "-l", "strimzi.io/cluster=kafka-main", "--for=condition=Ready", "--timeout=300s")
	if err != nil {
		return err
	}

	resources := []string{
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

	return platform.Kubernetes.Wait(ctx, messagingNamespace, "kafkaconnect.kafka.strimzi.io/kafka-connect", "condition=Ready", "900s")
}
