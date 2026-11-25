# Network and access

Language: **EN** | [RU](NETWORK.ru.md)

Back to [README](../README.md)

Run commands from `infrastructure/stand`.

| Namespace                                              | Contents                                                         |
| :----------------------------------------------------- | :--------------------------------------------------------------- |
| `kube-system`                                          | CoreDNS, metrics-server, and MicroK8s system components          |
| `openebs`                                              | Rawfile LocalPV                                                  |
| `istio-system`, `istio-ingress`                        | istiod, CNI, ztunnel, and public Gateway                         |
| `cert-manager`, `vault`, `vault-secrets-operator`      | PKI, Vault, and Vault secret delivery                            |
| `cnpg-system`, `strimzi-system`, `clickhouse-operator` | Data resource controllers                                        |
| `monadiam-apps`                                        | Five applications, their configuration, permissions, and secrets |
| `monadiam-data`                                        | PostgreSQL, Redis, ClickHouse, and SeaweedFS                     |
| `monadiam-messaging`                                   | Kafka, Connect, Apicurio, and shared schema publishing           |
| `monadiam-observability`                               | Prometheus stack, Loki, Tempo, and OTel                          |
| `argocd`                                               | Argo CD and Image Updater                                        |

Namespace labels are in [namespaces.yaml](../clusters/dev/infrastructure/namespaces/namespaces.yaml) and the [DEV overlay](../clusters/dev/infrastructure/namespaces/kustomization.yaml).

| Access                  | Address / port                                       | Purpose                                           |
| :---------------------- | :--------------------------------------------------- | :------------------------------------------------ |
| HTTP(S) by name         | `monadiam.localhost`, `*.monadiam.localhost`, 80/443 | Hostname routes. DNS must point to the stand host |
| AccessControl           | `https://<IP>:4001`                                  | API and service docs                              |
| Identity                | `https://<IP>:4002`                                  | Identity API and issuer                           |
| Notification            | `https://<IP>:4003`                                  | Notification API                                  |
| Organization            | `https://<IP>:4004`                                  | Organization API                                  |
| HR                      | `https://<IP>:4005`                                  | HR API                                            |
| Zot from host           | `http://127.0.0.1:5000`                              | OCI registry                                      |
| Zot for containerd/Pods | `monadiam-registry.monadiam-data.svc:5000`           | Image pulls and publishing                        |

Gateway uses the `istio` class. Each application HTTPRoute connects a listener to its Kubernetes Service. When accessing it from another machine, `localhost` refers to the client machine.

Istio ambient uses CNI and ztunnel. [STRICT mTLS](../clusters/dev/infrastructure/istio/base/peer-authentication.yaml) applies between enabled namespaces. PostgreSQL, Redis, and Kafka configure TLS separately. [NetworkPolicy](../clusters/dev/infrastructure/istio/base/network-policy.yaml) and application AuthorizationPolicies restrict access.

A local cert-manager CA issues certificates. Trust it on the client machine or use `curl --cacert`.

**Administrative interfaces**

```sh
just gitops::ui
just platform::grafana-ui
just platform::schema-ui
```

Each port-forward runs in its own terminal until Ctrl+C. Argo CD is at `https://127.0.0.1:8443` and Grafana at `http://127.0.0.1:3000`. Grafana credentials are in the `grafana-admin` Secret in `monadiam-observability`. If the initial Argo CD admin Secret still exists, read its password with:

```sh
kubectl --context "$KUBE_CONTEXT" -n argocd get secret argocd-initial-admin-secret \
  -o jsonpath='{.data.password}' | base64 --decode
```
