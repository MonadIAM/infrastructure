# @monadiam/stand-infrastructure

Platform | Stand Infrastructure: k3d/k3s, PostgreSQL/Redis, Kafka/Debezium messaging,
Vault secrets, ClickHouse, SeaweedFS archive storage, Go tooling and observability with Prometheus, Grafana,
Loki and Tempo.

### Requirements

- ***Make*** — runs the prepared project commands from the repository root.
- ***Go 1.27+*** — builds and runs the CLI, development checks and tests on the host.
- ***Docker*** — runs the k3s nodes managed by k3d; the Docker engine must be running.
- ***k3d*** — creates, starts and stops the Kubernetes cluster.
- ***kubectl and Helm*** — apply manifests, install charts and check readiness.
- ***Network access*** — required for container registries, Helm repositories and
  Gateway API release manifests during deployment.

----

<details>
<summary><strong>Host Ports</strong></summary>

| Component          | Host address       | Purpose                          |
| :----------------- | :----------------- | :------------------------------- |
| Kubernetes API     | `127.0.0.1:6550`   | Cluster management               |
| Gateway HTTP       | `127.0.0.1:8080`   | HTTP ingress                     |
| Gateway HTTPS      | `127.0.0.1:8443`   | HTTPS ingress                    |
| Container registry | `127.0.0.1:5001`   | Images for the local k3d cluster |

Bindings are defined in `bootstrap/k3d/cluster.yaml`. Gateway listeners use
`monadiam.localhost` and `*.monadiam.localhost`. Application routes and service
deployments must be supplied separately; creating the gateway does not deploy them.

</details>

----

<details>
<summary><strong>Structure</strong></summary>

```text
stand/
├── Makefile                            # Cluster, platform and development commands
├── .env.example                        # Environment settings and chart versions
├── main.go                             # CLI entry point and cancellation
├── go.mod                              # Go module and dependencies
├── go.sum                              # Dependency checksums
├── cli/                                # Go application and component tests
│   ├── cli.go                          # Command-line parsing
│   ├── commands.go                     # Command handlers and invocation state
│   ├── types.go                        # Command handler signature and usage
│   ├── config/                         # Dotenv loading, initialization and validation
│   ├── platform/
│   │   ├── command/                    # Subprocess execution and cancellable polling
│   │   ├── helm/                       # Chart installation and retries
│   │   ├── kubernetes/                 # Context selection, manifests and readiness
│   │   ├── seaweedfs/                  # S3 archive policies, versioning and retention
│   │   ├── stand/                      # Cluster lifecycle and platform reconciliation
│   │   │   └── definitions.go          # Phases, releases, operators and secrets
│   │   └── vault/                      # Initialization, unseal and bootstrap
│   │       └── definitions.go          # Credential and role definitions
│   └── tests/integration/              # Real CLI and isolated k3d lifecycle tests
│       ├── storage_helpers.go          # S3 client and archive checks
│       └── testdata/                   # Ephemeral ports and a dedicated image registry
├── bootstrap/k3d/cluster.yaml           # k3s nodes, registry and host bindings
├── clusters/local/                     # Namespaces and storage configuration
├── infrastructure/
│   ├── networking/                     # Istio and Gateway API
│   ├── security/                       # cert-manager and Vault Secrets Operator
│   ├── operators/                      # CloudNativePG, Strimzi and ClickHouse
│   ├── observability/                  # Prometheus, Grafana, Loki, Tempo and OTel
│   └── storage/                        # Retain storage class
└── platform/
    ├── postgresql/                     # CNPG cluster and monitoring resources
    ├── redis/                          # Redis chart values and TLS resources
    ├── kafka/                          # Kafka, Connect, topics and outbox connector
    ├── clickhouse/                     # ClickHouse cluster and monitoring resources
    ├── seaweedfs/                      # Helm values, bootstrap Job, bucket policies and monitoring
    ├── vault/                          # Server values, certificates and HCL policies
    └── secrets/                        # Secret contract
```

</details>

----

<details>
<summary><strong>Commands</strong></summary>

| Makefile            | CLI                                              | Description                                                                          |
| :------------------ | :----------------------------------------------- | :----------------------------------------------------------------------------------- |
| **Infrastructure**  |                                                  |                                                                                      |
| `make env`          | `infra config init`                              | Create a private environment file; preserve an existing file.                        |
| `make start`        | `infra cluster start`                            | Create or start the cluster and reconcile the platform.                              |
| `make resume`       | `infra cluster resume`                           | Start and reconcile an existing cluster; fail if absent.                             |
| `make stop`         | `infra cluster stop`                             | Stop the cluster without deleting it.                                                |
| `make restart`      | `infra cluster restart`                          | Stop, start and reconcile the existing cluster.                                      |
| `make recreate`     | `infra cluster recreate --confirm-delete`        | Delete and recreate the cluster; cluster data may be lost.                           |
| `make status`       | `infra cluster status`                           | Show cluster nodes, releases and platform resources.                                 |
| **Development**     |                                                  |                                                                                      |
| `make build`        | —                                                | Build the standalone CLI at `build/infra`.                                           |
| `make lint`         | —                                                | Run golangci-lint and gofmt checks.                                                  |
| `make lint-fix`     | —                                                | Apply gofmt formatting.                                                              |
| `make env-check`    | `infra config validate`                          | Validate environment settings without network calls.                                 |
| **Platform**        |                                                  |                                                                                      |
| `make reconcile`    | `infra platform reconcile`                       | Reconcile the platform in a running cluster.                                         |
| `make vault-status` | `infra vault status`                             | Show initialization, seal status and unseal threshold.                               |
| `make seaweedfs`    | `infra platform reconcile --component seaweedfs` | Reconcile SeaweedFS and configure archive buckets in an initialized platform.        |
| —                   | `infra seaweedfs reconcile`                      | Configure archive buckets; executed inside the bootstrap Job.                        |
| **Test**            |                                                  |                                                                                      |
| `make utest`        | —                                                | Run Ginkgo tests with the race detector, without k3s.                                |
| `make itest`        | —                                                | Test bootstrap, S3 access, retention and data persistence in a separate k3d cluster. |

| CLI option         | Purpose                                                         |
| :----------------- | :-------------------------------------------------------------- |
| `--env-file`       | Dotenv path; `--env-file=` disables file loading.               |
| `--timeout`        | Total operation timeout, e.g. `60m`; not used by `config init`. |
| `--component`      | Platform component to reconcile; defaults to `all`.             |
| `--confirm-delete` | Allow cluster deletion, only for `cluster recreate`.            |
| `--help`           | General or command-specific help.                               |

</details>

----
