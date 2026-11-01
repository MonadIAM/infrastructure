auth_mount = "kubernetes"
audience   = "vault"
role_ttl   = 3600
namespace_bindings = {
  monadiam-apps          = { policy = "monadiam-apps-read", role = "monadiam-apps" }
  monadiam-data          = { policy = "monadiam-data-read", role = "monadiam-data" }
  monadiam-messaging     = { policy = "monadiam-messaging-read", role = "monadiam-messaging" }
  monadiam-observability = { policy = "monadiam-observability-read", role = "monadiam-observability" }
  clickhouse-operator    = { policy = "clickhouse-operator-read", role = "clickhouse-operator" }
}
