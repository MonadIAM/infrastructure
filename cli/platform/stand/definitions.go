package stand

import (
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

func (platform Platform) phases() []phaseDefinition {
	phases := []phaseDefinition{
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
			Name: "vault",
			Run:  platform.Vault,
		},
		{
			Name: "gateway",
			Run:  platform.Gateway,
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
			Name: "seaweedfs",
			Run:  platform.SeaweedFS,
		},
	}

	return phases
}

func releaseDefinitions(environment config.Environment) map[string]helm.Release {
	profile := "clusters/" + environment["PROFILE"] + "/infrastructure"
	releases := map[string]helm.Release{
		"seaweedfs-main": {
			Name:      "seaweedfs-main",
			Chart:     "seaweedfs/seaweedfs",
			Version:   environment["SEAWEEDFS_CHART_VERSION"],
			Namespace: dataNamespace,
			Values:    "infrastructure/seaweedfs/values.yaml",
			Arguments: []string{"--values", profile + "/seaweedfs/values.yaml", "--wait", "--timeout", "600s"},
		},
		"istio-base": {
			Name:      "istio-base",
			Chart:     "istio/base",
			Version:   environment["ISTIO_VERSION"],
			Namespace: "istio-system",
			Values:    "infrastructure/istio/base-values.yaml",
			Arguments: []string{"--values", profile + "/istio/values.yaml"},
		},
		"istiod": {
			Name:      "istiod",
			Chart:     "istio/istiod",
			Version:   environment["ISTIO_VERSION"],
			Namespace: "istio-system",
			Values:    "infrastructure/istio/istiod-values.yaml",
			Arguments: []string{"--values", profile + "/istio/values.yaml", "--values", profile + "/istio/istiod-values.yaml"},
		},
		"istio-cni": {
			Name:      "istio-cni",
			Chart:     "istio/cni",
			Version:   environment["ISTIO_VERSION"],
			Namespace: "istio-system",
			Values:    "infrastructure/istio/istio-cni-values.yaml",
			Arguments: []string{"--values", profile + "/istio/values.yaml", "--values", profile + "/istio/cni-values.yaml"},
		},
		"ztunnel": {
			Name:      "ztunnel",
			Chart:     "istio/ztunnel",
			Version:   environment["ISTIO_VERSION"],
			Namespace: "istio-system",
			Values:    "infrastructure/istio/ztunnel-values.yaml",
			Arguments: []string{"--values", profile + "/istio/values.yaml", "--values", profile + "/istio/ztunnel-values.yaml"},
		},
		"vault-secrets-operator": {
			Name:      "vault-secrets-operator",
			Chart:     "hashicorp/vault-secrets-operator",
			Version:   environment["VAULT_SECRETS_OPERATOR_VERSION"],
			Namespace: vaultOperatorNamespace,
			Values:    "infrastructure/vault-secrets-operator/values.yaml",
			Arguments: []string{"--values", profile + "/vault-secrets-operator/values.yaml"},
		},
	}

	for _, cluster := range redisClusters {
		releases[cluster] = helm.Release{
			Name:      cluster,
			Chart:     environment["REDIS_CHART"],
			Version:   environment["REDIS_CHART_VERSION"],
			Namespace: dataNamespace,
			Values:    "infrastructure/redis/values.yaml",
			Arguments: []string{
				"--values", "infrastructure/redis/services/" + strings.TrimSuffix(cluster, "-redis") + ".yaml",
				"--values", profile + "/redis/values.yaml",
				"--values", profile + "/redis/" + strings.TrimSuffix(cluster, "-redis") + ".yaml",
				"--wait", "--timeout", "600s", "--hide-notes",
			},
		}
	}

	return releases
}

func operatorDefinitions() []operatorDefinition {
	return []operatorDefinition{
		{
			Namespace:  "cnpg-system",
			Deployment: "cnpg-cloudnative-pg",
			CRDs:       []string{"clusters.postgresql.cnpg.io"},
		},
		{
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
