path "{{kv_mount}}/data/{{clickhouse_path}}" {
    capabilities = ["read"]
}

path "{{kv_mount}}/data/{{minio_path}}" {
    capabilities = ["read"]
}
