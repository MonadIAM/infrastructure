path "kv/data/monadiam-data/postgresql/template/app" {
    capabilities = ["read"]
}

path "kv/data/monadiam-data/redis/template/auth" {
    capabilities = ["read"]
}

path "transit/hmac/template-audit-mask-hmac" {
    capabilities = ["update"]
}

path "transit/keys/template-audit-log-sig" {
    capabilities = ["read"]
}

path "transit/sign/template-audit-log-sig" {
    capabilities = ["update"]
}

path "transit/verify/template-audit-log-sig" {
    capabilities = ["update"]
}
