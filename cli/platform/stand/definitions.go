package stand

const kafkaResources = "kafka.kafka.strimzi.io,kafkanodepool.kafka.strimzi.io,kafkatopic.kafka.strimzi.io,kafkauser.kafka.strimzi.io,kafkaconnect.kafka.strimzi.io,kafkaconnector.kafka.strimzi.io"

const (
	appsNamespace           = "monadiam-apps"
	dataNamespace           = "monadiam-data"
	messagingNamespace      = "monadiam-messaging"
	observabilityNamespace  = "monadiam-observability"
	vaultOperatorNamespace  = "vault-secrets-operator"
	vaultOperatorDeployment = "deployment/vault-secrets-operator-controller-manager"
)

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
