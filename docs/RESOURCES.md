# Resources and disks

Language: **EN** | [RU](RESOURCES.ru.md)

Back to [README](../README.md)

The values below apply to the `dev` profile. Check actual requirements for a new host before installation.

[monadiam.slice](../clusters/dev/host/monadiam.slice) sets a shared resource limit:

| Setting         | Value                          |
| :-------------- | :----------------------------- |
| `CPUQuota`      | `1200%`, equivalent to 12 CPUs |
| `MemoryHigh`    | 22 GB                          |
| `MemoryMax`     | 24 GB, decimal units           |
| `MemorySwapMax` | 0                              |

The slice includes MicroK8s, Pods, Zot, and the current host's Redis backup service. The OS, Docker daemon, and user-started builds are accounted for separately. Container limits can add up to more than the slice limit.

**Applications and storage**

CPU/RAM figures are per instance. Requests affect Pod scheduling, while limits constrain containers. An application rolling update may temporarily add one replica per Deployment.

| Component                  | Count               | Requests     | Limits       | Storage / detail                         |
| :------------------------- | :------------------ | :----------- | :----------- | :--------------------------------------- |
| Node.js application        | 5                   | 100m / 256Mi | 2 / 1Gi      | 1 GiB uploads each, Node.js heap 768 MiB |
| Application Vault Agent    | 5                   | 25m / 64Mi   | 100m / 128Mi | Permanent native sidecar                 |
| PostgreSQL instance        | 5                   | 1 / 1Gi      | 1 / 1Gi      | 16Gi data + 1Gi WAL each                 |
| Redis                      | 5                   | 100m / 128Mi | 500m / 512Mi | 1Gi AOF/RDB each, `maxmemory 128mb`      |
| Sentinel                   | 5                   | 25m / 32Mi   | 200m / 64Mi  | 100Mi each                               |
| Redis exporter             | 5                   | 25m / 64Mi   | 200m / 128Mi | No dedicated PVC                         |
| Kafka broker/controller    | 1                   | 250m / 768Mi | 1 / 1536Mi   | 5Gi, 768m heap                           |
| Kafka topic operator       | 1                   | 100m / 256Mi | 500m / 512Mi | 128-256m heap                            |
| Kafka user operator        | 1                   | 100m / 256Mi | 500m / 512Mi | 128-256m heap                            |
| Connect worker             | 1                   | 100m / 512Mi | 1 / 1Gi      | 256-512m heap, offsets in Kafka          |
| Connect build              | Per build           | 100m / 512Mi | 1 / 1Gi      | Publishes image to Zot                   |
| Apicurio                   | 1                   | 100m / 512Mi | 1 / 1Gi      | 128-512m heap, data in Kafka             |
| Vault server               | 1                   | 50m / 256Mi  | 1 / 512Mi    | 2Gi Raft                                 |
| ClickHouse                 | 1                   | 250m / 768Mi | 1 / 1536Mi   | 5Gi                                      |
| SeaweedFS master           | 1                   | 100m / 128Mi | 250m / 512Mi | 1Gi                                      |
| SeaweedFS volume           | 1                   | 100m / 256Mi | 500m / 1Gi   | 20Gi                                     |
| SeaweedFS filer/S3         | 1                   | 100m / 256Mi | 500m / 1Gi   | 2Gi metadata                             |
| shared-schemas Job         | Per sync            | 50m / 32Mi   | 500m / 128Mi | 32Mi tmpfs                               |
| SeaweedFS bootstrap Job    | Per reconcile       | 50m / 32Mi   | 500m / 128Mi | Configures buckets and lifecycle         |
| ClickHouse schema Job      | Per reconcile       | 50m / 64Mi   | 500m / 256Mi | Applies SQL schema                       |
| ClickHouse archive CronJob | Daily               | 50m / 64Mi   | 500m / 256Mi | Parquet in SeaweedFS, no PVC             |
| Istio Gateway proxy        | Per Gateway replica | 100m / 128Mi | 1 / 384Mi    | Created automatically by Istio           |
| Zot Deployment             | 1                   | See manifest | See manifest | Data on a host volume                    |

**Controllers and monitoring**

Requests and limits include DEV overrides. Section names come from Helm values. `main` means the component's own resources. Hooks and init containers are temporary. A DaemonSet runs on each eligible node.

| Component / Helm values section                                                                                                 | Requests CPU / RAM | Limits CPU / RAM |
| :------------------------------------------------------------------------------------------------------------------------------ | :----------------- | :--------------- |
| [cert-manager](../clusters/dev/infrastructure/cert-manager/operator/values.yaml) / `main`                                       | 10m / 64Mi         | 250m / 192Mi     |
| [cert-manager](../clusters/dev/infrastructure/cert-manager/operator/values.yaml) / `cainjector`                                 | 10m / 64Mi         | 250m / 192Mi     |
| [cert-manager](../clusters/dev/infrastructure/cert-manager/operator/values.yaml) / `webhook`                                    | 10m / 32Mi         | 250m / 128Mi     |
| [cert-manager](../clusters/dev/infrastructure/cert-manager/operator/values.yaml) / `startupapicheck`                            | 10m / 32Mi         | 250m / 128Mi     |
| [CNPG](../clusters/dev/infrastructure/postgresql/operator/values.yaml) / `main`                                                 | 50m / 128Mi        | 500m / 384Mi     |
| [Strimzi](../clusters/dev/infrastructure/kafka/operator/values.yaml) / `main`                                                   | 200m / 384Mi       | 1 / 384Mi        |
| [ClickHouse operator](../clusters/dev/infrastructure/clickhouse/operator/values.yaml) / `crdHook`                               | 10m / 32Mi         | 250m / 128Mi     |
| [ClickHouse operator](../clusters/dev/infrastructure/clickhouse/operator/values.yaml) / `operator`                              | 25m / 64Mi         | 250m / 192Mi     |
| [ClickHouse operator](../clusters/dev/infrastructure/clickhouse/operator/values.yaml) / `metrics`                               | 10m / 64Mi         | 200m / 128Mi     |
| [VSO](../clusters/dev/infrastructure/vault/operator/values.yaml) / `controller.manager`                                         | 10m / 64Mi         | 500m / 256Mi     |
| [VSO](../clusters/dev/infrastructure/vault/operator/values.yaml) / `controller.kubeRbacProxy`                                   | 5m / 64Mi          | 500m / 128Mi     |
| [VSO](../clusters/dev/infrastructure/vault/operator/values.yaml) / `hooks`                                                      | 10m / 64Mi         | 500m / 128Mi     |
| [ESO](../clusters/dev/infrastructure/external-secrets/operator-values.yaml) / `main`                                            | 25m / 64Mi         | 250m / 256Mi     |
| [ESO](../clusters/dev/infrastructure/external-secrets/operator-values.yaml) / `webhook`                                         | 10m / 32Mi         | 100m / 128Mi     |
| [ESO](../clusters/dev/infrastructure/external-secrets/operator-values.yaml) / `certController`                                  | 10m / 32Mi         | 100m / 128Mi     |
| [Reloader](../clusters/dev/infrastructure/reloader/values.yaml) / `reloader.deployment`                                         | 10m / 32Mi         | 100m / 128Mi     |
| [Rawfile](../clusters/dev/infrastructure/rawfile/values.yaml) / `node.driverRegistrar`                                          | 10m / 16Mi         | 100m / 64Mi      |
| [Rawfile](../clusters/dev/infrastructure/rawfile/values.yaml) / `node.externalProvisioner`                                      | 10m / 32Mi         | 100m / 128Mi     |
| [Rawfile](../clusters/dev/infrastructure/rawfile/values.yaml) / `controller`                                                    | 25m / 64Mi         | 250m / 192Mi     |
| [Rawfile](../clusters/dev/infrastructure/rawfile/values.yaml) / `controller.externalResizer`                                    | 10m / 32Mi         | 100m / 64Mi      |
| [istiod](../clusters/dev/infrastructure/istio/istiod-values.yaml) / `main`                                                      | 100m / 128Mi       | 500m / 384Mi     |
| [Istio CNI](../clusters/dev/infrastructure/istio/cni-values.yaml) / `main`                                                      | 100m / 100Mi       | 250m / 256Mi     |
| [ztunnel](../clusters/dev/infrastructure/istio/ztunnel-values.yaml) / `main`                                                    | 100m / 128Mi       | 750m / 256Mi     |
| [Argo CD](../clusters/dev/infrastructure/argocd/values.yaml) / `controller`                                                     | 100m / 256Mi       | 500m / 1Gi       |
| [Argo CD](../clusters/dev/infrastructure/argocd/values.yaml) / `repoServer`                                                     | 100m / 128Mi       | 500m / 512Mi     |
| [Argo CD](../clusters/dev/infrastructure/argocd/values.yaml) / `server`                                                         | 50m / 128Mi        | 250m / 256Mi     |
| [Argo CD](../clusters/dev/infrastructure/argocd/values.yaml) / `redis`                                                          | 25m / 64Mi         | 100m / 128Mi     |
| [Argo CD](../clusters/dev/infrastructure/argocd/values.yaml) / `redisSecretInit`                                                | 10m / 32Mi         | 250m / 128Mi     |
| [Image Updater](../clusters/dev/infrastructure/image-updater/values.yaml) / `main`                                              | 50m / 64Mi         | 250m / 256Mi     |
| [Prometheus stack](../clusters/dev/infrastructure/prometheus-stack/values.yaml) / `alertmanager.alertmanagerSpec`               | 25m / 64Mi         | 100m / 128Mi     |
| [Grafana](../clusters/dev/infrastructure/grafana/values.yaml) / `resources`                                                     | 50m / 128Mi        | 250m / 384Mi     |
| [Grafana](../clusters/dev/infrastructure/grafana/values.yaml) / `sidecar`                                                       | 10m / 64Mi         | 100m / 128Mi     |
| [Grafana](../clusters/dev/infrastructure/grafana/values.yaml) / `initChownData`                                                 | 10m / 16Mi         | 100m / 64Mi      |
| [Prometheus stack](../clusters/dev/infrastructure/prometheus-stack/values.yaml) / `prometheusOperator`                          | 50m / 128Mi        | 250m / 256Mi     |
| [Prometheus stack](../clusters/dev/infrastructure/prometheus-stack/values.yaml) / `prometheusOperator.prometheusConfigReloader` | 10m / 32Mi         | 100m / 128Mi     |
| [Prometheus stack](../clusters/dev/infrastructure/prometheus-stack/values.yaml) / `prometheusOperator.admissionWebhooks.patch`  | 10m / 32Mi         | 250m / 128Mi     |
| [Prometheus stack](../clusters/dev/infrastructure/prometheus-stack/values.yaml) / `prometheus.prometheusSpec`                   | 100m / 512Mi       | 500m / 1536Mi    |
| [kube-state-metrics](../clusters/dev/infrastructure/kube-state-metrics/values.yaml) / `resources`                               | 25m / 64Mi         | 250m / 256Mi     |
| [node-exporter](../clusters/dev/infrastructure/node-exporter/values.yaml) / `resources`                                         | 10m / 32Mi         | 200m / 64Mi      |
| [Loki](../clusters/dev/infrastructure/loki/values.yaml) / `singleBinary`                                                        | 50m / 256Mi        | 500m / 768Mi     |
| [Loki](../clusters/dev/infrastructure/loki/values.yaml) / `sidecar`                                                             | 10m / 64Mi         | 100m / 128Mi     |
| [Tempo](../clusters/dev/infrastructure/tempo/values.yaml) / `tempo`                                                             | 50m / 256Mi        | 500m / 1536Mi    |
| [OTel](../clusters/dev/infrastructure/otel-collector/values.yaml) / `main`                                                      | 50m / 128Mi        | 500m / 512Mi     |

In `kube-system`, LimitRange supplies missing requests of 5m / 16Mi and limits of 100m / 64Mi. CoreDNS limits are 250m / 170Mi, and metrics-server limits are 250m / 192Mi.

**Disks and PVCs**

| Area                | Host file / path                                                                  | Configured size      | Contents                                                      |
| :------------------ | :-------------------------------------------------------------------------------- | :------------------- | :------------------------------------------------------------ |
| MicroK8s            | Data path depends on the MicroK8s installation                                    | Host-dependent       | Cluster state, containerd, kubelet, and Pod logs              |
| Main PVC pool       | `/var/lib/monadiam/microk8s-volumes.ext4` -> `/var/lib/monadiam/microk8s/volumes` | Host-dependent       | Rawfile `local-retain` volumes                                |
| PostgreSQL WAL pool | `/var/lib/monadiam/postgresql-wal`                                                | No shared size limit | Rawfile `local-wal-retain` WAL volumes on the host filesystem |
| Zot                 | `/var/lib/monadiam/zot.ext4` -> `/var/lib/monadiam/zot`                           | 32Gi                 | Registry data                                                 |
| Terraform           | `TF_STATE_DIR/<profile>/<context>/<root>`                                         | No separate quota    | State and provider data                                       |
| Vault keys          | `VAULT_STATE_DIR`, `VAULT_INIT_FILE`                                              | No separate quota    | Unseal keys, root token, and CLI CA                           |

| Data                                | Requested size                 |
| :---------------------------------- | :----------------------------- |
| PostgreSQL data                     | 5 × 16Gi                       |
| PostgreSQL WAL                      | 5 × 1Gi in a separate WAL pool |
| Redis data / Sentinel               | 5 × 1Gi / 5 × 100Mi            |
| Kafka                               | 5Gi                            |
| Vault                               | 2Gi                            |
| ClickHouse                          | 5Gi                            |
| SeaweedFS master / volume / filer   | 1Gi / 20Gi / 2Gi               |
| Prometheus / Alertmanager / Grafana | 2Gi / 1Gi / 1Gi                |
| Loki / Tempo                        | 2Gi / see DEV values           |
| Application uploads                 | 5 × 1Gi                        |

The current host's volumes do not define requirements for a new machine. Allow separate space for WAL, the OS, Docker images/build cache, and state. The WAL pool is directly on the host filesystem, outside the Rawfile volume.

Both storage classes use ext4, `Retain`, `WaitForFirstConsumer`, and `allowVolumeExpansion`. Thin provisioning and copy-on-write are disabled. Local PVs are bound to the node. Both pools reserve 5% free space.

```sh
kubectl --context "$KUBE_CONTEXT" get pvc -A
kubectl --context "$KUBE_CONTEXT" get pv
findmnt /var/lib/monadiam/microk8s/volumes
df -h /var/lib/monadiam/microk8s/volumes /var/lib/monadiam/zot
sudo du -sh /var/lib/monadiam/postgresql-wal
```

DEV overrides set PVC sizes. Before expansion, check free space in the pool and on the host. Host volume preparation and resizing are separate from stand startup.

**Retention and rotation**

| Object            | Setting                                                                                                         |
| :---------------- | :-------------------------------------------------------------------------------------------------------------- |
| Prometheus        | 24h retention and 1500MB retentionSize on a 2Gi PVC                                                             |
| Alertmanager      | 24h retention                                                                                                   |
| Loki              | 24h retention, filesystem, 2Gi PVC                                                                              |
| Tempo             | 48h retention, 2Gi PVC, ballast 0, `GOMEMLIMIT=1200MiB`, container limit 1536Mi                                 |
| Pod logs          | Kubelet: 10Mi × 3 files per container                                                                           |
| MicroK8s logs     | Logging settings depend on the host installation                                                                |
| Zot               | Hourly GC with 1h delay, 24h retention delay, last five pushed tags, `keep-(dev\|prod)-*`, untagged rules       |
| SeaweedFS buckets | `audit-archive`: 1095 days, `change-log-archive`: 365 days                                                      |
| Kafka             | Retention is set per KafkaTopic. See [topics](../clusters/dev/infrastructure/kafka/base/kafka-main-topics.yaml) |
