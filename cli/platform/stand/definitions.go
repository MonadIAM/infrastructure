package stand

import (
	"github.com/MonadIAM/infrastructure/cli/config"
	"github.com/MonadIAM/infrastructure/cli/platform/helm"
)

const kafkaResources = "kafka.kafka.strimzi.io,kafkanodepool.kafka.strimzi.io,kafkatopic.kafka.strimzi.io,kafkauser.kafka.strimzi.io,kafkaconnect.kafka.strimzi.io,kafkaconnector.kafka.strimzi.io"

const (
	dataNamespace           = "monadiam-data"
	messagingNamespace      = "monadiam-messaging"
	observabilityNamespace  = "monadiam-observability"
	vaultOperatorNamespace  = "vault-secrets-operator"
	vaultOperatorDeployment = "deployment/vault-secrets-operator-controller-manager"
)

func (platform Platform) phases() []phaseDefinition {
	return []phaseDefinition{
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
			Name: "observability",
			Run:  platform.Observability,
		},
	}
}

func releaseDefinitions(environment config.Environment) map[string]helm.Release {
	return map[string]helm.Release{
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
			Arguments: []string{"--set", "platform=" + environment["ISTIO_PLATFORM"]},
		},
		"istiod": {
			Name:      "istiod",
			Chart:     "istio/istiod",
			Version:   environment["ISTIO_VERSION"],
			Namespace: "istio-system",
			Values:    "infrastructure/networking/istio/helm/istiod-values.yaml",
			Arguments: []string{"--set", "platform=" + environment["ISTIO_PLATFORM"]},
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
		"redis-main": {
			Name:      "redis-main",
			Chart:     environment["REDIS_CHART"],
			Version:   environment["REDIS_CHART_VERSION"],
			Namespace: dataNamespace,
			Values:    "platform/redis/helm/values.yaml",
			Arguments: []string{"--wait", "--timeout", "600s", "--hide-notes"},
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
			Namespace:   "monadiam-data",
			Name:        "postgresql-main-superuser",
			Destination: "postgresql-main-superuser",
		},
		{
			Namespace:   "monadiam-data",
			Name:        "postgresql-main-app",
			Destination: "postgresql-main-app",
		},
		{
			Namespace:   "monadiam-data",
			Name:        "postgresql-main-debezium",
			Destination: "postgresql-main-debezium",
		},
		{
			Namespace:   "monadiam-messaging",
			Name:        "kafka-main-admin-password",
			Destination: "kafka-main-admin-password",
		},
		{
			Namespace:   "monadiam-data",
			Name:        "redis-main-auth",
			Destination: "redis-main-auth",
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
