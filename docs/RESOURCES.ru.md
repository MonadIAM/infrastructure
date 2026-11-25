# Ресурсы и диски

Language: [EN](RESOURCES.md) | **RU**

Вернуться к [README](../README.ru.md)

Параметры ниже относятся к профилю `dev`. Фактические требования новой машины проверяются до установки.

[monadiam.slice](../clusters/dev/host/monadiam.slice) задаёт общий лимит ресурсов:

| Параметр        | Значение                   |
| :-------------- | :------------------------- |
| `CPUQuota`      | `1200%`, эквивалент 12 CPU |
| `MemoryHigh`    | 22 GB                      |
| `MemoryMax`     | 24 GB, десятичные единицы  |
| `MemorySwapMax` | 0                          |

В slice входят MicroK8s, Pod, Zot и Redis backup service текущего хоста.
ОС, Docker daemon и запущенные пользователем сборки учитываются отдельно.
Сумма container limits может превышать общий предел slice.

**Приложения и хранилища**

Значения CPU/RAM указаны на один экземпляр. Requests влияют на размещение Pod,
limits ограничивают контейнер. Во время rolling update приложений допускается
одна дополнительная реплика на Deployment.

| Компонент                  | Количество         | Requests     | Limits       | Хранилище/параметр                     |
| :------------------------- | :----------------- | :----------- | :----------- | :------------------------------------- |
| Приложение Node.js         | 5                  | 100m / 256Mi | 2 / 1Gi      | По 1 GiB uploads, Node.js heap 768 MiB |
| Vault Agent приложений     | 5                  | 25m / 64Mi   | 100m / 128Mi | Постоянный native sidecar              |
| PostgreSQL instance        | 5                  | 1 / 1Gi      | 1 / 1Gi      | По 16Gi data + 1Gi WAL                 |
| Redis                      | 5                  | 100m / 128Mi | 500m / 512Mi | По 1Gi AOF/RDB, `maxmemory 128mb`      |
| Sentinel                   | 5                  | 25m / 32Mi   | 200m / 64Mi  | По 100Mi                               |
| Redis exporter             | 5                  | 25m / 64Mi   | 200m / 128Mi | Без своего PVC                         |
| Kafka broker/controller    | 1                  | 250m / 768Mi | 1 / 1536Mi   | 5Gi, heap 768m                         |
| Kafka topic operator       | 1                  | 100m / 256Mi | 500m / 512Mi | Heap 128–256m                          |
| Kafka user operator        | 1                  | 100m / 256Mi | 500m / 512Mi | Heap 128–256m                          |
| Connect worker             | 1                  | 100m / 512Mi | 1 / 1Gi      | Heap 256–512m, offsets в Kafka         |
| Connect build              | На сборку          | 100m / 512Mi | 1 / 1Gi      | Публикация образа в Zot                |
| Apicurio                   | 1                  | 100m / 512Mi | 1 / 1Gi      | Heap 128–512m, данные в Kafka          |
| Vault server               | 1                  | 50m / 256Mi  | 1 / 512Mi    | 2Gi Raft                               |
| ClickHouse                 | 1                  | 250m / 768Mi | 1 / 1536Mi   | 5Gi                                    |
| SeaweedFS master           | 1                  | 100m / 128Mi | 250m / 512Mi | 1Gi                                    |
| SeaweedFS volume           | 1                  | 100m / 256Mi | 500m / 1Gi   | 20Gi                                   |
| SeaweedFS filer/S3         | 1                  | 100m / 256Mi | 500m / 1Gi   | 2Gi metadata                           |
| shared-schemas Job         | При sync           | 50m / 32Mi   | 500m / 128Mi | 32Mi tmpfs                             |
| SeaweedFS bootstrap Job    | При reconcile      | 50m / 32Mi   | 500m / 128Mi | Настройка buckets/lifecycle            |
| ClickHouse schema Job      | При reconcile      | 50m / 64Mi   | 500m / 256Mi | Применение SQL схемы                   |
| ClickHouse archive CronJob | Раз в сутки        | 50m / 64Mi   | 500m / 256Mi | Parquet в SeaweedFS, без PVC           |
| Istio Gateway proxy        | На реплику Gateway | 100m / 128Mi | 1 / 384Mi    | Автоматически создаётся Istio          |
| Zot Deployment             | 1                  | См. manifest | См. manifest | Данные на томе хоста                   |

**Контроллеры и monitoring**

В таблице приведены requests и limits с учётом DEV overrides.
Имена секций взяты из Helm values, «основной» означает ресурсы самого компонента. Hooks и init-контейнеры работают
временно, DaemonSet создаёт экземпляр на каждом подходящем узле.

| Компонент / секция Helm values                                                                                                  | Requests CPU / RAM | Limits CPU / RAM |
| :------------------------------------------------------------------------------------------------------------------------------ | :----------------- | :--------------- |
| [cert-manager](../clusters/dev/infrastructure/cert-manager/operator/values.yaml) / `основной`                                   | 10m / 64Mi         | 250m / 192Mi     |
| [cert-manager](../clusters/dev/infrastructure/cert-manager/operator/values.yaml) / `cainjector`                                 | 10m / 64Mi         | 250m / 192Mi     |
| [cert-manager](../clusters/dev/infrastructure/cert-manager/operator/values.yaml) / `webhook`                                    | 10m / 32Mi         | 250m / 128Mi     |
| [cert-manager](../clusters/dev/infrastructure/cert-manager/operator/values.yaml) / `startupapicheck`                            | 10m / 32Mi         | 250m / 128Mi     |
| [CNPG](../clusters/dev/infrastructure/postgresql/operator/values.yaml) / `основной`                                             | 50m / 128Mi        | 500m / 384Mi     |
| [Strimzi](../clusters/dev/infrastructure/kafka/operator/values.yaml) / `основной`                                               | 200m / 384Mi       | 1 / 384Mi        |
| [ClickHouse operator](../clusters/dev/infrastructure/clickhouse/operator/values.yaml) / `crdHook`                               | 10m / 32Mi         | 250m / 128Mi     |
| [ClickHouse operator](../clusters/dev/infrastructure/clickhouse/operator/values.yaml) / `operator`                              | 25m / 64Mi         | 250m / 192Mi     |
| [ClickHouse operator](../clusters/dev/infrastructure/clickhouse/operator/values.yaml) / `metrics`                               | 10m / 64Mi         | 200m / 128Mi     |
| [VSO](../clusters/dev/infrastructure/vault/operator/values.yaml) / `controller.manager`                                         | 10m / 64Mi         | 500m / 256Mi     |
| [VSO](../clusters/dev/infrastructure/vault/operator/values.yaml) / `controller.kubeRbacProxy`                                   | 5m / 64Mi          | 500m / 128Mi     |
| [VSO](../clusters/dev/infrastructure/vault/operator/values.yaml) / `hooks`                                                      | 10m / 64Mi         | 500m / 128Mi     |
| [ESO](../clusters/dev/infrastructure/external-secrets/operator-values.yaml) / `основной`                                        | 25m / 64Mi         | 250m / 256Mi     |
| [ESO](../clusters/dev/infrastructure/external-secrets/operator-values.yaml) / `webhook`                                         | 10m / 32Mi         | 100m / 128Mi     |
| [ESO](../clusters/dev/infrastructure/external-secrets/operator-values.yaml) / `certController`                                  | 10m / 32Mi         | 100m / 128Mi     |
| [Reloader](../clusters/dev/infrastructure/reloader/values.yaml) / `reloader.deployment`                                         | 10m / 32Mi         | 100m / 128Mi     |
| [Rawfile](../clusters/dev/infrastructure/rawfile/values.yaml) / `node.driverRegistrar`                                          | 10m / 16Mi         | 100m / 64Mi      |
| [Rawfile](../clusters/dev/infrastructure/rawfile/values.yaml) / `node.externalProvisioner`                                      | 10m / 32Mi         | 100m / 128Mi     |
| [Rawfile](../clusters/dev/infrastructure/rawfile/values.yaml) / `controller`                                                    | 25m / 64Mi         | 250m / 192Mi     |
| [Rawfile](../clusters/dev/infrastructure/rawfile/values.yaml) / `controller.externalResizer`                                    | 10m / 32Mi         | 100m / 64Mi      |
| [istiod](../clusters/dev/infrastructure/istio/istiod-values.yaml) / `основной`                                                  | 100m / 128Mi       | 500m / 384Mi     |
| [Istio CNI](../clusters/dev/infrastructure/istio/cni-values.yaml) / `основной`                                                  | 100m / 100Mi       | 250m / 256Mi     |
| [ztunnel](../clusters/dev/infrastructure/istio/ztunnel-values.yaml) / `основной`                                                | 100m / 128Mi       | 750m / 256Mi     |
| [Argo CD](../clusters/dev/infrastructure/argocd/values.yaml) / `controller`                                                     | 100m / 256Mi       | 500m / 1Gi       |
| [Argo CD](../clusters/dev/infrastructure/argocd/values.yaml) / `repoServer`                                                     | 100m / 128Mi       | 500m / 512Mi     |
| [Argo CD](../clusters/dev/infrastructure/argocd/values.yaml) / `server`                                                         | 50m / 128Mi        | 250m / 256Mi     |
| [Argo CD](../clusters/dev/infrastructure/argocd/values.yaml) / `redis`                                                          | 25m / 64Mi         | 100m / 128Mi     |
| [Argo CD](../clusters/dev/infrastructure/argocd/values.yaml) / `redisSecretInit`                                                | 10m / 32Mi         | 250m / 128Mi     |
| [Image Updater](../clusters/dev/infrastructure/image-updater/values.yaml) / `основной`                                          | 50m / 64Mi         | 250m / 256Mi     |
| [Prometheus stack](../clusters/dev/infrastructure/prometheus-stack/values.yaml) / `alertmanager.alertmanagerSpec`               | 25m / 64Mi         | 100m / 128Mi     |
| [Grafana](../clusters/dev/infrastructure/grafana/values.yaml) / `resources`                                                     | 50m / 128Mi        | 250m / 384Mi     |
| [Grafana](../clusters/dev/infrastructure/grafana/values.yaml) / `sidecar`                                                       | 10m / 64Mi         | 100m / 128Mi     |
| [Grafana](../clusters/dev/infrastructure/grafana/values.yaml) / `initChownData`                                                 | 10m / 16Mi         | 100m / 64Mi      |
| [Prometheus stack](../clusters/dev/infrastructure/prometheus-stack/values.yaml) / `prometheusOperator`                          | 50m / 128Mi        | 250m / 256Mi     |
| [Prometheus stack](../clusters/dev/infrastructure/prometheus-stack/values.yaml) / `prometheusOperator.prometheusConfigReloader` | 10m / 32Mi         | 100m / 64Mi      |
| [Prometheus stack](../clusters/dev/infrastructure/prometheus-stack/values.yaml) / `prometheusOperator.admissionWebhooks.patch`  | 10m / 32Mi         | 250m / 128Mi     |
| [Prometheus stack](../clusters/dev/infrastructure/prometheus-stack/values.yaml) / `prometheus.prometheusSpec`                   | 100m / 512Mi       | 500m / 1536Mi    |
| [kube-state-metrics](../clusters/dev/infrastructure/kube-state-metrics/values.yaml) / `resources`                               | 25m / 64Mi         | 250m / 256Mi     |
| [node-exporter](../clusters/dev/infrastructure/node-exporter/values.yaml) / `resources`                                         | 10m / 32Mi         | 200m / 64Mi      |
| [Loki](../clusters/dev/infrastructure/loki/values.yaml) / `singleBinary`                                                        | 50m / 256Mi        | 500m / 768Mi     |
| [Loki](../clusters/dev/infrastructure/loki/values.yaml) / `sidecar`                                                             | 10m / 64Mi         | 100m / 128Mi     |
| [Tempo](../clusters/dev/infrastructure/tempo/values.yaml) / `tempo`                                                             | 50m / 256Mi        | 500m / 1536Mi    |
| [OTel](../clusters/dev/infrastructure/otel-collector/values.yaml) / `основной`                                                  | 50m / 128Mi        | 500m / 512Mi     |

В `kube-system` LimitRange задаёт отсутствующие requests 5m / 16Mi
и limits 100m / 64Mi. У CoreDNS limits 250m / 170Mi,
у metrics-server — 250m / 192Mi.

**Диски и PVC**

| Область             | Файл/путь хоста                                                                   | Размер по конфигурации    | Что хранит                                                       |
| :------------------ | :-------------------------------------------------------------------------------- | :------------------------ | :--------------------------------------------------------------- |
| MicroK8s            | Путь данных зависит от установки MicroK8s                                         | На хосте                  | Cluster state, containerd, kubelet и Pod logs                    |
| Основной пул PVC    | `/var/lib/monadiam/microk8s-volumes.ext4` -> `/var/lib/monadiam/microk8s/volumes` | На хосте                  | Rawfile volumes `local-retain`                                   |
| PostgreSQL WAL pool | `/var/lib/monadiam/postgresql-wal`                                                | Общий размер не ограничен | Rawfile WAL volumes `local-wal-retain` на файловой системе хоста |
| Zot                 | `/var/lib/monadiam/zot.ext4` -> `/var/lib/monadiam/zot`                           | 32Gi                      | Registry data                                                    |
| Terraform           | `TF_STATE_DIR/<profile>/<context>/<root>`                                         | Без отдельной квоты       | State и provider data                                            |
| Ключи Vault         | `VAULT_STATE_DIR`, `VAULT_INIT_FILE`                                              | Без отдельной квоты       | Unseal keys, root token и CA для CLI                             |

| Данные                              | Запрошенный размер           |
| :---------------------------------- | :--------------------------- |
| PostgreSQL data                     | 5 × 16Gi                     |
| PostgreSQL WAL                      | 5 × 1Gi в отдельном WAL pool |
| Redis data / Sentinel               | 5 × 1Gi / 5 × 100Mi          |
| Kafka                               | 5Gi                          |
| Vault                               | 2Gi                          |
| ClickHouse                          | 5Gi                          |
| SeaweedFS master / volume / filer   | 1Gi / 20Gi / 2Gi             |
| Prometheus / Alertmanager / Grafana | 2Gi / 1Gi / 1Gi              |
| Loki / Tempo                        | 2Gi / см. DEV values         |
| Uploads приложений                  | 5 × 1Gi                      |

Размеры томов текущего хоста не задают требований для новой машины. Место для
WAL, ОС, Docker images/build cache и state требуется отдельно. WAL pool
находится прямо на файловой системе хоста, вне тома Rawfile.

Оба storage class используют ext4, `Retain`, `WaitForFirstConsumer`
и `allowVolumeExpansion`. Thin provisioning и copy-on-write отключены.
Local PV привязан к узлу. В обоих пулах зарезервировано 5% свободного места.

```sh
kubectl --context "$KUBE_CONTEXT" get pvc -A
kubectl --context "$KUBE_CONTEXT" get pv
findmnt /var/lib/monadiam/microk8s/volumes
df -h /var/lib/monadiam/microk8s/volumes /var/lib/monadiam/zot
sudo du -sh /var/lib/monadiam/postgresql-wal
```

Размеры PVC задаются в DEV overrides. Перед расширением проверьте свободное место
в пуле и на хосте. Подготовка и изменение размера хостовых томов выполняются
отдельно от запуска стенда.

**Retention и ротация**

| Объект            | Настройка                                                                                                                          |
| :---------------- | :--------------------------------------------------------------------------------------------------------------------------------- |
| Prometheus        | Retention 24h и retentionSize 1500MB на PVC 2Gi                                                                                    |
| Alertmanager      | Retention 24h                                                                                                                      |
| Loki              | Retention 24h, filesystem, PVC 2Gi                                                                                                 |
| Tempo             | Retention 48h, PVC 2Gi, ballast 0, `GOMEMLIMIT=1200MiB`, container limit 1536Mi                                                    |
| Pod logs          | Kubelet: 10Mi × 3 файла на контейнер                                                                                               |
| Журналы MicroK8s  | Настройки журналирования зависят от установки на хосте                                                                             |
| Zot               | GC каждый час с delay 1h, retention delay 24h, последние пять pushed tags, `keep-(dev\|prod)-*`, правила untagged                  |
| SeaweedFS buckets | `audit-archive`: 1095 дней, `change-log-archive`: 365 дней                                                                         |
| Kafka             | Retention задаётся отдельно для каждого KafkaTopic, см. [topics](../clusters/dev/infrastructure/kafka/base/kafka-main-topics.yaml) |
