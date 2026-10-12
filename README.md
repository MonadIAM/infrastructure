# @monadiam/stand-infrastructure

Platform | Stand Infrastructure: k3d/k3s, PostgreSQL/Redis, Kafka/Debezium messaging,
Vault secrets, ClickHouse, SeaweedFS archive storage, Go tooling and observability
with Prometheus, Grafana, Loki and Tempo.

> If your machine has insufficient CPU or memory, switch to the `local` branch.
> It provides a less resource-intensive setup using Docker Compose.

### Requirements

- **Make** — runs the prepared project commands from `Makefile`, such as `make start`.
- **Go 1.27+** — builds and runs the CLI, checks and tests on the host.
- **Docker** — runs containers with k3s nodes; Docker must be running.
- **k3d** — creates and manages the Kubernetes cluster based on k3s.
- **kubectl and Helm** — apply manifests, install components and check readiness.
- **Network access** — required to download images, Helm charts and Gateway API manifests.

### Quick Start

Run from the repository root:

```bash
make env
```

Review `.env`, especially `CLUSTER`, `KUBE_CONTEXT` and `VAULT_INIT_FILE`.
Then validate the configuration and start the stand:

```bash
make env-check
make start
make status
```

`make start` creates or starts the cluster, installs platform components and waits
for them to become ready. Applications and their routes are deployed separately.

> - `make env` preserves an existing `.env`.
> - Use `make resume` to start a stopped stand and `make stop` to stop it.
> - `make recreate` deletes the cluster before creating a new one; data may be lost.

----

<details>
<summary><strong>Commands</strong></summary>

| Makefile            | CLI                                              | Purpose                                                 |
| :------------------ | :----------------------------------------------- | :------------------------------------------------------ |
| **Infrastructure**  |                                                  |                                                         |
| `make env`          | `infra config init`                              | Create `.env` if it does not exist.                     |
| `make start`        | `infra cluster start`                            | Create or start the cluster and configure the platform. |
| `make resume`       | `infra cluster resume`                           | Start and configure an existing cluster.                |
| `make stop`         | `infra cluster stop`                             | Stop the cluster without deleting it.                   |
| `make restart`      | `infra cluster restart`                          | Restart the cluster and apply configuration.            |
| `make recreate`     | `infra cluster recreate --confirm-delete`        | Delete and recreate the cluster.                        |
| `make status`       | `infra cluster status`                           | Show nodes, releases and resources.                     |
| **Development**     |                                                  |                                                         |
| `make build`        | —                                                | Build the executable at `build/infra`.                  |
| `make lint`         | —                                                | Run golangci-lint and gofmt checks.                     |
| `make lint-fix`     | —                                                | Apply gofmt formatting.                                 |
| `make env-check`    | `infra config validate`                          | Validate settings without network calls.                |
| **Platform**        |                                                  |                                                         |
| `make reconcile`    | `infra platform reconcile`                       | Update the running platform configuration.              |
| `make vault-status` | `infra vault status`                             | Show Vault status and the unseal key threshold.         |
| `make seaweedfs`    | `infra platform reconcile --component seaweedfs` | Apply SeaweedFS and bucket configuration.               |
| —                   | `infra seaweedfs reconcile`                      | Configure buckets inside the bootstrap Job.             |
| **Tests**           |                                                  |                                                         |
| `make utest`        | —                                                | Run Ginkgo tests with the race detector, without k3s.   |
| `make itest`        | —                                                | Test the platform in an isolated k3d cluster.           |

**CLI Options**

| Option             | Purpose                                                |
| :----------------- | :----------------------------------------------------- |
| `--env-file`       | Dotenv path; an empty value disables file loading.     |
| `--timeout`        | Total timeout, e.g. `60m`; not used by `config init`.  |
| `--component`      | Component for `platform reconcile`; defaults to `all`. |
| `--confirm-delete` | Allow deletion for `cluster recreate`.                 |
| `--help`           | General or command-specific help.                      |

Components available for selective reconciliation: `cert-manager`, `metrics-server`,
`istio`, `gateway`, `vault`, `operators`, `clusters`, `seaweedfs`, `observability`.

> - Makefile is the main entry point. To run the CLI directly, use
>   `go run . <component> <action> [options]`; for help, use `go run . --help`.
> - Select an environment file through `ENV_FILE`:
>   `make env-check ENV_FILE=.env.example`.
> - `make reconcile COMPONENT=seaweedfs` applies only the selected component;
>   its dependencies must already be configured by a full platform startup.
> - Single-quote dotenv values containing `$`.
> - Integration tests cover initialization, S3 permissions, repeated reconciliation
>   and restarts with data persistence. The test cluster and registry are removed;
>   the JSON report is saved to `build/integration-report.json`.

</details>

----

<details>
<summary><strong>Structure</strong></summary>

```text
stand/
├── Makefile                             # Cluster, platform and development commands
├── .env.example                         # Environment settings and component versions
├── main.go                              # CLI entry point and shutdown handling
├── go.mod                               # Go module and dependencies
├── go.sum                               # Dependency checksums
│
├── cli/                                 # Go application and component tests
│   ├── cli.go                           # Command-line arguments
│   ├── commands.go                      # Command handlers and application state
│   ├── types.go                         # Handler type and help text
│   ├── config/                          # Configuration loading, creation and validation
│   ├── platform/
│   │   ├── command/                     # Subprocess execution and readiness polling
│   │   ├── helm/                        # Chart installation and retries
│   │   ├── kubernetes/                  # Context, manifests and resource readiness
│   │   ├── seaweedfs/                   # Buckets, versioning and retention rules
│   │   ├── stand/                       # Cluster and platform component management
│   │   └── vault/                       # Initialization, unseal and secret configuration
│   └── tests/integration/               # Tests against a real k3d cluster
│       ├── cluster_helpers.go           # Test environment management
│       ├── platform_helpers.go          # Resource state, Vault and PostgreSQL checks
│       ├── seaweedfs_helpers.go         # S3 settings and port forwarding
│       ├── storage_helpers.go           # S3 client and object checks
│       ├── bootstrap_test.go            # Bootstrap, permissions and data persistence
│       └── testdata/                    # Isolated ports and test registry
│
├── bootstrap/k3d/cluster.yaml           # k3s nodes, registry and host ports
├── clusters/local/                      # Namespaces and storage configuration
├── infrastructure/
│   ├── networking/                      # Istio and Gateway API
│   ├── security/                        # cert-manager and Vault Secrets Operator
│   ├── operators/                       # CloudNativePG, Strimzi and ClickHouse Operator
│   ├── observability/                   # Prometheus, Grafana, Loki, Tempo and OTel
│   └── storage/                         # Storage class with the Retain policy
└── platform/
    ├── postgresql/                      # CNPG cluster and monitoring
    ├── redis/                           # Helm values and TLS certificates
    ├── kafka/                           # Broker, Connect, topics and outbox connector
    ├── clickhouse/                      # ClickHouse cluster and monitoring
    ├── seaweedfs/                       # Helm values, bootstrap Job and S3 policies
    ├── vault/                           # Server, certificates and access policies
    └── secrets/contract.yaml            # Vault paths and corresponding Secrets
```

</details>

----

<details>
<summary><strong>Ports and Access</strong></summary>

Host bindings are defined in `bootstrap/k3d/cluster.yaml`.

| Component      | Host address     | Purpose                     |
| :------------- | :--------------- | :-------------------------- |
| Kubernetes API | `127.0.0.1:6550` | Cluster management.         |
| Gateway HTTP   | `127.0.0.1:8080` | HTTP ingress.               |
| Gateway HTTPS  | `127.0.0.1:8443` | HTTPS ingress.              |
| Registry       | `127.0.0.1:5001` | Images for the k3d cluster. |

The Gateway accepts `monadiam.localhost` and `*.monadiam.localhost`.
Application `HTTPRoute` resources from `monadiam-apps` are added separately.
Creating the Gateway does not expose Grafana, Vault or other interfaces.

For local access, forward a port from the required resource. Grafana example
using the default context:

```bash
kubectl --context k3d-monadiam -n monadiam-observability \
  port-forward deployment/kube-prometheus-stack-grafana 3000:3000
```

Open `http://localhost:3000` while the command is running. If you changed
`KUBE_CONTEXT` in `.env`, use that value instead of `k3d-monadiam`.

</details>

----

<details>
<summary><strong>Service Infrastructure</strong></summary>

By default, k3d creates one k3s server node and one worker node.
Platform components are organized into namespaces:

| Namespace                | Components                                      |
| :----------------------- | :---------------------------------------------- |
| `monadiam-data`          | PostgreSQL, Redis, ClickHouse and SeaweedFS.    |
| `monadiam-messaging`     | Kafka and Kafka Connect.                        |
| `monadiam-observability` | Metrics, logs, traces and Grafana.              |
| `vault`                  | Vault server.                                   |
| `monadiam-apps`          | Namespace for applications deployed separately. |

- **PostgreSQL:** CloudNativePG manages `postgresql-main`. It creates the `app` database,
  an application user and the `debezium` replication role.
- **Redis:** the Helm chart configures replication, Sentinel, password authentication
  and TLS with client certificate verification.
- **Storage:** persistent data uses `local-path-retain`. The Retain policy is not a
  backup and does not guarantee data preservation when the entire k3d cluster is deleted.

Manifests are stored in `platform/<component>/`, and operators in
`infrastructure/operators/`. `make start` does not deploy applications.

</details>

----

<details>
<summary><strong>Vault</strong></summary>

During platform setup, the CLI installs, initializes and unseals Vault, creates the
KV store and configures Kubernetes authentication. After a restart, the CLI unseals
Vault using the saved keys.

| Setting               | Purpose                                               |
| :-------------------- | :---------------------------------------------------- |
| `VAULT_STATE_DIR`     | Vault state directory on the machine running the CLI. |
| `VAULT_INIT_FILE`     | JSON file containing the root token and unseal keys.  |
| `VAULT_KEY_SHARES`    | Number of key shares created during initialization.   |
| `VAULT_KEY_THRESHOLD` | Number of key shares required to unseal Vault.        |

By default, keys and the root token are written to `/tmp/monadiam/k3s/vault/init.json`.
Directory permissions are set to `0700`, and file permissions to `0600`.

Vault Secrets Operator copies data from KV v2 into Kubernetes Secrets.
Secret paths, fields, access roles and consuming components are described in
`platform/secrets/contract.yaml`; access policies are stored in `platform/vault/bootstrap/`.
Existing credentials are preserved when configuration is reapplied.

> `VAULT_INIT_FILE` is required to unseal Vault again. For a long-lived stand,
> set `VAULT_STATE_DIR` and `VAULT_INIT_FILE` to locations outside `/tmp`.

</details>

----

<details>
<summary><strong>Kafka and Debezium</strong></summary>

Strimzi manages Kafka, Kafka Connect, topics, users and connectors in the
`monadiam-messaging` namespace. The internal Kafka listener uses TLS and
SCRAM authentication.

| File in `platform/kafka/resources/`     | Purpose                                              |
| :-------------------------------------- | :--------------------------------------------------- |
| `kafka-main.yaml`                       | Broker and listener settings.                        |
| `kafka-main-topics.yaml`                | Topics and retention settings.                       |
| `kafka-connect.yaml`                    | Connect and its image build with Debezium.           |
| `postgresql-main-outbox-connector.yaml` | Connector for `system.outbox` in the `app` database. |

The Connect image is built with the PostgreSQL Debezium plugin and published to
the cluster registry. The connector reads PostgreSQL credentials from a Kubernetes Secret.

> - The `postgresql-main-outbox` connector is declared with `state: stopped`.
>   Prepare the `system.outbox` table and change the state in the manifest before enabling it.
> - Schema Registry is not deployed in the current `stand` configuration.

</details>

----

<details>
<summary><strong>Audit Storage</strong></summary>

ClickHouse Operator deploys `clickhouse-main`, and the SeaweedFS Helm chart deploys
S3 storage. A bootstrap Job configures buckets, versioning, Object Lock in
COMPLIANCE mode and lifecycle rules.

| Default bucket       | Retention | Retention variable                    |
| :------------------- | :-------- | :------------------------------------ |
| `audit-archive`      | 1095 days | `SEAWEEDFS_AUDIT_LOG_RETENTION_DAYS`  |
| `change-log-archive` | 365 days  | `SEAWEEDFS_CHANGE_LOG_RETENTION_DAYS` |

Bucket names are set through `SEAWEEDFS_AUDIT_LOG_BUCKET` and
`SEAWEEDFS_CHANGE_LOG_BUCKET`. The administrator and archiver use separate credentials,
provided by Vault through Kubernetes Secrets. The archiver can read and write objects;
bucket policies prohibit deletion.

`make seaweedfs` reapplies storage configuration and runs the bootstrap Job.
Policy templates are stored in `platform/seaweedfs/config/`.

> The current `stand` deploys ClickHouse and S3 storage. The audit SQL schema and
> periodic partition export from `local` are not yet included in the deployment.

</details>

----

<details>
<summary><strong>Observability</strong></summary>

Components are deployed in the `monadiam-observability` namespace;
Helm settings are stored in `infrastructure/observability/`.

| Component      | Purpose                                                         |
| :------------- | :-------------------------------------------------------------- |
| Prometheus     | Kubernetes and component metrics via ServiceMonitor/PodMonitor. |
| Alertmanager   | Prometheus alert handling.                                      |
| Grafana        | Dashboards and Prometheus, Loki and Tempo datasources.          |
| Loki           | Log ingestion and storage.                                      |
| Tempo          | Trace storage.                                                  |
| OTel Collector | OTLP ingestion and trace forwarding to Tempo.                   |

By default, Prometheus retains metrics for 24 hours. Node-exporter, kube-state-metrics
and Kubernetes API monitoring are enabled. Platform components have their own
monitoring resources.

To access Grafana, use the port forwarding command in "Ports and Access".
The Grafana username and password are stored in Vault and copied to the `grafana-admin` Secret.

> - Applications must configure telemetry export and metrics endpoints separately.
> - OTel Collector writes received OTLP metrics only to its diagnostic log.
>   Prometheus collects metrics by scraping components; log ingestion through
>   OTel Collector is not configured.

</details>
