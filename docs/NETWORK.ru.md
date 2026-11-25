# Сеть и доступ

Language: [EN](NETWORK.md) | **RU**

Вернуться к [README](../README.ru.md)

Команды выполняются из каталога `infrastructure/stand`.

| Область                                                | Содержимое                                              |
| :----------------------------------------------------- | :------------------------------------------------------ |
| `kube-system`                                          | CoreDNS, metrics-server и системные компоненты MicroK8s |
| `openebs`                                              | Rawfile LocalPV                                         |
| `istio-system`, `istio-ingress`                        | istiod, CNI, ztunnel, публичный Gateway                 |
| `cert-manager`, `vault`, `vault-secrets-operator`      | PKI, Vault и доставка секретов из Vault                 |
| `cnpg-system`, `strimzi-system`, `clickhouse-operator` | Контроллеры ресурсов данных                             |
| `monadiam-apps`                                        | Пять приложений, их конфигурация, права и секреты       |
| `monadiam-data`                                        | PostgreSQL, Redis, ClickHouse, SeaweedFS                |
| `monadiam-messaging`                                   | Kafka, Connect, Apicurio, публикация shared schemas     |
| `monadiam-observability`                               | Prometheus stack, Loki, Tempo, OTel                     |
| `argocd`                                               | Argo CD и Image Updater                                 |

Метки namespaces задаются в [namespaces.yaml](../clusters/dev/infrastructure/namespaces/namespaces.yaml)
и [DEV overlay](../clusters/dev/infrastructure/namespaces/kustomization.yaml).

| Доступ                 | Адрес/порт                                           | Назначение                                                |
| :--------------------- | :--------------------------------------------------- | :-------------------------------------------------------- |
| HTTP(S) по имени       | `monadiam.localhost`, `*.monadiam.localhost`, 80/443 | Маршруты по hostname, DNS должен указывать на хост стенда |
| AccessControl          | `https://<IP>:4001`                                  | API и документация сервиса                                |
| Identity               | `https://<IP>:4002`                                  | Identity API, issuer                                      |
| Notification           | `https://<IP>:4003`                                  | Notification API                                          |
| Organization           | `https://<IP>:4004`                                  | Organization API                                          |
| HR                     | `https://<IP>:4005`                                  | HR API                                                    |
| Zot с хоста            | `http://127.0.0.1:5000`                              | OCI registry                                              |
| Zot для containerd/Pod | `monadiam-registry.monadiam-data.svc:5000`           | Загрузка и публикация образов                             |

Gateway использует класс `istio`. HTTPRoute каждого приложения связывает listener
с его Kubernetes Service. При доступе с другой машины `localhost` относится к машине клиента.

Istio ambient использует CNI и ztunnel. Между включёнными namespaces действует
[STRICT mTLS](../clusters/dev/infrastructure/istio/base/peer-authentication.yaml).
TLS PostgreSQL, Redis и Kafka настраивается отдельно.
Доступ ограничивают [NetworkPolicy](../clusters/dev/infrastructure/istio/base/network-policy.yaml)
и AuthorizationPolicy приложений.

Сертификаты выпускает локальный CA cert-manager. Добавьте его в доверенные
на машине клиента или передавайте через `curl --cacert`.

**Административные интерфейсы**

```sh
just gitops::ui
just platform::grafana-ui
just platform::schema-ui
```

Каждый port-forward запускается в отдельном терминале и работает до Ctrl+C.
Argo CD доступен на `https://127.0.0.1:8443`, Grafana — на `http://127.0.0.1:3000`.
Grafana credentials находятся в Secret `grafana-admin` namespace `monadiam-observability`.
Начальный пароль администратора Argo CD, если Secret ещё существует:

```sh
kubectl --context "$KUBE_CONTEXT" -n argocd get secret argocd-initial-admin-secret \
  -o jsonpath='{.data.password}' | base64 --decode
```
