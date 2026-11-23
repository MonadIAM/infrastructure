path "kv/data/monadiam-messaging/bootstrap" {
  capabilities = ["read"]
}

path "kv/metadata/monadiam-messaging/bootstrap" {
  capabilities = ["read"]
}

path "kv/data/monadiam-messaging/kafka/*" {
  capabilities = ["read"]
}

path "kv/metadata/monadiam-messaging/kafka/*" {
  capabilities = ["read"]
}

path "kv/data/monadiam-messaging/kafka-connect/*" {
  capabilities = ["read"]
}

path "kv/metadata/monadiam-messaging/kafka-connect/*" {
  capabilities = ["read"]
}
