package stand

import (
	"context"
	"strings"

	"github.com/MonadIAM/infrastructure/cli/config"
	"github.com/MonadIAM/infrastructure/cli/platform/helm"
)

const kafkaResources = "kafka.kafka.strimzi.io,kafkanodepool.kafka.strimzi.io,kafkatopic.kafka.strimzi.io,kafkauser.kafka.strimzi.io,kafkaconnect.kafka.strimzi.io,kafkaconnector.kafka.strimzi.io"

const (
	appsNamespace           = "monadiam-apps"
	dataNamespace           = "monadiam-data"
	messagingNamespace      = "monadiam-messaging"
	observabilityNamespace  = "monadiam-observability"
	vaultOperatorNamespace  = "vault-secrets-operator"
	vaultOperatorDeployment = "deployment/vault-secrets-operator-controller-manager"
)

var istioComponents = []struct {
	Release  string
	Workload string
}{
	{Release: "istio-base"},
	{Release: "istiod", Workload: "deployment/istiod"},
	{Release: "istio-cni", Workload: "daemonset/istio-cni-node"},
	{Release: "ztunnel", Workload: "daemonset/ztunnel"},
}

var kafkaPartitionKeys = []string{
	"PROJECTION_JOURNAL_PARTITIONS",
	"REALM_PARTITIONS",
	"POSITION_PARTITIONS",
}

var postgresqlClusters = []string{
	"access-control-postgresql",
	"identity-postgresql",
	"notification-postgresql",
	"organization-postgresql",
	"hr-postgresql",
}

var redisClusters = []string{
	"access-control-redis",
	"identity-redis",
	"notification-redis",
	"organization-redis",
	"hr-redis",
}

var applications = []struct {
	Name     string
	ImageKey string
	Port     string
}{
	{Name: "access-control", ImageKey: "ACCESS_CONTROL_IMAGE", Port: "4001"},
	{Name: "identity", ImageKey: "IDENTITY_IMAGE", Port: "4002"},
	{Name: "notification", ImageKey: "NOTIFICATION_IMAGE", Port: "4003"},
	{Name: "organization", ImageKey: "ORGANIZATION_IMAGE", Port: "4004"},
	{Name: "hr", ImageKey: "HR_IMAGE", Port: "4005"},
}

func (platform Platform) phases() []phaseDefinition {
	phases := []phaseDefinition{
		{
			Name: "images",
			Run:  platform.Images,
		},
		{
			Name: "cert-manager",
			Run:  platform.CertManager,
		},
		{
			Name: "metrics-server",
			Run:  platform.MetricsServer,
		},
		{
			Name: "istio",
			Run:  platform.Istio,
		},
		{
			Name: "gateway",
			Run:  platform.Gateway,
		},
		{
			Name: "vault",
			Run:  platform.Vault,
		},
		{
			Name: "operators",
			Run:  platform.Operators,
		},
		{
			Name: "clusters",
			Run:  platform.Clusters,
		},
		{
			Name: "apicurio",
			Run:  platform.Apicurio,
		},
		{
			Name: "schemas",
			Run:  platform.Schemas,
		},
		{
			Name: "seaweedfs",
			Run:  platform.SeaweedFS,
		},
		{
			Name: "observability",
			Run:  platform.Observability,
		},
	}

	for _, application := range applications {
		phases = append(phases, phaseDefinition{Name: application.Name, Run: func(ctx context.Context) error {
			return platform.Application(ctx, application.Name, application.ImageKey)
		}})
	}

	return phases
}

func releaseDefinitions(environment config.Environment) map[string]helm.Release {
	releases := map[string]helm.Release{
		"seaweedfs-main": {
			Name:      "seaweedfs-main",
			Chart:     "seaweedfs/seaweedfs",
			Version:   environment["SEAWEEDFS_CHART_VERSION"],
			Namespace: dataNamespace,
			Values:    "platform/seaweedfs/helm/values.yaml",
			Arguments: []string{"--set-string", "image.tag=" + environment["SEAWEEDFS_VERSION"], "--wait", "--timeout", "600s"},
		},
		"cert-manager": {
			Name:      "cert-manager",
			Chart:     environment["CERT_MANAGER_CHART"],
			Version:   environment["CERT_MANAGER_VERSION"],
			Namespace: "cert-manager",
			Values:    "infrastructure/security/cert-manager/helm/values.yaml",
		},
		"istio-base": {
			Name:      "istio-base",
			Chart:     "istio/base",
			Version:   environment["ISTIO_VERSION"],
			Namespace: "istio-system",
			Values:    "infrastructure/networking/istio/helm/base-values.yaml",
			Arguments: []string{"--set", "global.platform=" + environment["ISTIO_PLATFORM"]},
		},
		"istiod": {
			Name:      "istiod",
			Chart:     "istio/istiod",
			Version:   environment["ISTIO_VERSION"],
			Namespace: "istio-system",
			Values:    "infrastructure/networking/istio/helm/istiod-values.yaml",
			Arguments: []string{"--set", "global.platform=" + environment["ISTIO_PLATFORM"]},
		},
		"istio-cni": {
			Name:      "istio-cni",
			Chart:     "istio/cni",
			Version:   environment["ISTIO_VERSION"],
			Namespace: "istio-system",
			Values:    "infrastructure/networking/istio/helm/istio-cni-values.yaml",
			Arguments: []string{"--set", "global.platform=" + environment["ISTIO_PLATFORM"]},
		},
		"ztunnel": {
			Name:      "ztunnel",
			Chart:     "istio/ztunnel",
			Version:   environment["ISTIO_VERSION"],
			Namespace: "istio-system",
			Values:    "infrastructure/networking/istio/helm/ztunnel-values.yaml",
			Arguments: []string{"--set", "global.platform=" + environment["ISTIO_PLATFORM"]},
		},
		"vault": {
			Name:      "vault",
			Chart:     "hashicorp/vault",
			Version:   environment["VAULT_CHART_VERSION"],
			Namespace: environment["VAULT_NAMESPACE"],
			Values:    "platform/vault/helm/values.yaml",
		},
		"vault-secrets-operator": {
			Name:      "vault-secrets-operator",
			Chart:     "hashicorp/vault-secrets-operator",
			Version:   environment["VAULT_SECRETS_OPERATOR_VERSION"],
			Namespace: vaultOperatorNamespace,
			Values:    "infrastructure/security/vault-secrets-operator/helm/values.yaml",
		},
		"cnpg": {
			Name:      "cnpg",
			Chart:     "cnpg/cloudnative-pg",
			Version:   environment["CNPG_CHART_VERSION"],
			Namespace: "cnpg-system",
			Values:    "infrastructure/operators/cnpg/helm/values.yaml",
		},
		"strimzi": {
			Name:      "strimzi",
			Chart:     "oci://quay.io/strimzi-helm/strimzi-kafka-operator",
			Version:   environment["STRIMZI_CHART_VERSION"],
			Namespace: "strimzi-system",
			Values:    "infrastructure/operators/strimzi/helm/values.yaml",
		},
		"clickhouse-operator": {
			Name:      "clickhouse-operator",
			Chart:     "altinity/altinity-clickhouse-operator",
			Version:   environment["CLICKHOUSE_OPERATOR_CHART_VERSION"],
			Namespace: "clickhouse-operator",
			Values:    "infrastructure/operators/clickhouse/helm/values.yaml",
		},
		"kube-prometheus-stack": {
			Name:      "kube-prometheus-stack",
			Chart:     "prometheus-community/kube-prometheus-stack",
			Version:   environment["KUBE_PROMETHEUS_STACK_CHART_VERSION"],
			Namespace: observabilityNamespace,
			Values:    "infrastructure/observability/prometheus-stack/helm/values.yaml",
			Arguments: []string{"--skip-crds", "--timeout", "600s"},
		},
		"loki": {
			Name:      "loki",
			Chart:     "grafana-community/loki",
			Version:   environment["LOKI_CHART_VERSION"],
			Namespace: observabilityNamespace,
			Values:    "infrastructure/observability/loki/helm/values.yaml",
			Arguments: []string{"--timeout", "600s"},
		},
		"tempo": {
			Name:      "tempo",
			Chart:     "grafana-community/tempo",
			Version:   environment["TEMPO_CHART_VERSION"],
			Namespace: observabilityNamespace,
			Values:    "infrastructure/observability/tempo/helm/values.yaml",
			Arguments: []string{"--timeout", "600s"},
		},
		"otel-collector": {
			Name:      "otel-collector",
			Chart:     "open-telemetry/opentelemetry-collector",
			Version:   environment["OPENTELEMETRY_COLLECTOR_CHART_VERSION"],
			Namespace: observabilityNamespace,
			Values:    "infrastructure/observability/otel-collector/helm/values.yaml",
			Arguments: []string{"--timeout", "600s"},
		},
	}

	for _, cluster := range redisClusters {
		releases[cluster] = helm.Release{
			Name:      cluster,
			Chart:     environment["REDIS_CHART"],
			Version:   environment["REDIS_CHART_VERSION"],
			Namespace: dataNamespace,
			Values:    "platform/redis/helm/values.yaml",
			Arguments: []string{
				"--values", "platform/redis/helm/services/" + strings.TrimSuffix(cluster, "-redis") + ".yaml",
				"--wait", "--timeout", "600s", "--hide-notes",
			},
		}
	}

	return releases
}

func operatorDefinitions() []operatorDefinition {
	return []operatorDefinition{
		{
			Release:    "cnpg",
			Namespace:  "cnpg-system",
			Deployment: "cnpg-cloudnative-pg",
			CRDs:       []string{"clusters.postgresql.cnpg.io"},
		},
		{
			Release:    "strimzi",
			Namespace:  "strimzi-system",
			Deployment: "strimzi-cluster-operator",
			CRDs: []string{
				"kafkas.kafka.strimzi.io",
				"kafkanodepools.kafka.strimzi.io",
				"kafkaconnects.kafka.strimzi.io",
				"kafkaconnectors.kafka.strimzi.io",
				"kafkausers.kafka.strimzi.io",
				"kafkatopics.kafka.strimzi.io",
			},
		},
		{
			Release:    "clickhouse-operator",
			Namespace:  "clickhouse-operator",
			Deployment: "clickhouse-operator-altinity-clickhouse-operator",
			Restart:    true,
			CRDs: []string{
				"clickhouseinstallations.clickhouse.altinity.com",
				"clickhousekeeperinstallations.clickhouse-keeper.altinity.com",
			},
		},
	}
}

func secretDefinitions() []secretDefinition {
	return []secretDefinition{
		{
			Namespace:   dataNamespace,
			Name:        "access-control-postgresql-app",
			Destination: "access-control-postgresql-app",
		},
		{
			Namespace:   dataNamespace,
			Name:        "access-control-postgresql-debezium",
			Destination: "access-control-postgresql-debezium",
		},
		{
			Namespace:   dataNamespace,
			Name:        "identity-postgresql-app",
			Destination: "identity-postgresql-app",
		},
		{
			Namespace:   dataNamespace,
			Name:        "identity-postgresql-debezium",
			Destination: "identity-postgresql-debezium",
		},
		{
			Namespace:   dataNamespace,
			Name:        "notification-postgresql-app",
			Destination: "notification-postgresql-app",
		},
		{
			Namespace:   dataNamespace,
			Name:        "notification-postgresql-debezium",
			Destination: "notification-postgresql-debezium",
		},
		{
			Namespace:   dataNamespace,
			Name:        "organization-postgresql-app",
			Destination: "organization-postgresql-app",
		},
		{
			Namespace:   dataNamespace,
			Name:        "organization-postgresql-debezium",
			Destination: "organization-postgresql-debezium",
		},
		{
			Namespace:   dataNamespace,
			Name:        "hr-postgresql-app",
			Destination: "hr-postgresql-app",
		},
		{
			Namespace:   dataNamespace,
			Name:        "hr-postgresql-debezium",
			Destination: "hr-postgresql-debezium",
		},
		{
			Namespace:   dataNamespace,
			Name:        "seaweedfs-main-s3",
			Destination: "seaweedfs-main-s3",
		},
		{
			Namespace:   dataNamespace,
			Name:        "seaweedfs-main-admin",
			Destination: "seaweedfs-main-admin",
		},
		{
			Namespace:   dataNamespace,
			Name:        "seaweedfs-main-archiver",
			Destination: "seaweedfs-main-archiver",
		},
		{
			Namespace:   "monadiam-apps",
			Name:        "monadiam-bootstrap",
			Destination: "monadiam-bootstrap",
		},
		{
			Namespace:   "monadiam-data",
			Name:        "monadiam-data-bootstrap",
			Destination: "monadiam-data-bootstrap",
		},
		{
			Namespace:   "monadiam-messaging",
			Name:        "monadiam-messaging-bootstrap",
			Destination: "monadiam-messaging-bootstrap",
		},
		{
			Namespace:   "monadiam-observability",
			Name:        "grafana-admin",
			Destination: "grafana-admin",
		},
		{
			Namespace:   "monadiam-messaging",
			Name:        "kafka-main-admin-password",
			Destination: "kafka-main-admin-password",
		},
		{
			Namespace:   "monadiam-data",
			Name:        "access-control-redis-auth",
			Destination: "access-control-redis-auth",
		},
		{
			Namespace:   "monadiam-data",
			Name:        "identity-redis-auth",
			Destination: "identity-redis-auth",
		},
		{
			Namespace:   "monadiam-data",
			Name:        "notification-redis-auth",
			Destination: "notification-redis-auth",
		},
		{
			Namespace:   "monadiam-data",
			Name:        "organization-redis-auth",
			Destination: "organization-redis-auth",
		},
		{
			Namespace:   "monadiam-data",
			Name:        "hr-redis-auth",
			Destination: "hr-redis-auth",
		},
		{
			Namespace:   "monadiam-data",
			Name:        "clickhouse-main-app",
			Destination: "clickhouse-main-app",
		},
		{
			Namespace:   "monadiam-data",
			Name:        "clickhouse-main-intercluster",
			Destination: "clickhouse-main-intercluster",
		},
		{
			Namespace:   "clickhouse-operator",
			Name:        "clickhouse-operator-credentials",
			Destination: "clickhouse-operator-altinity-clickhouse-operator",
		},
	}
}
