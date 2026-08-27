path "kv/data/monadiam-data/bootstrap" {
  capabilities = ["read"]
}

path "kv/metadata/monadiam-data/bootstrap" {
  capabilities = ["read"]
}

path "kv/data/monadiam-data/postgresql/*" {
  capabilities = ["read"]
}

path "kv/metadata/monadiam-data/postgresql/*" {
  capabilities = ["read"]
}

path "kv/data/monadiam-data/redis/*" {
  capabilities = ["read"]
}

path "kv/metadata/monadiam-data/redis/*" {
  capabilities = ["read"]
}

path "kv/data/monadiam-data/clickhouse/*" {
  capabilities = ["read"]
}

path "kv/metadata/monadiam-data/clickhouse/*" {
  capabilities = ["read"]
}

path "kv/data/monadiam-data/minio/*" {
  capabilities = ["read"]
}

path "kv/metadata/monadiam-data/minio/*" {
  capabilities = ["read"]
}
