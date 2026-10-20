path "kv/data/monadiam-data/postgresql/hr/app" {
    capabilities = ["read"]
}

path "kv/data/monadiam-data/redis/hr/auth" {
    capabilities = ["read"]
}

path "transit/hmac/hr-audit-mask-hmac" {
    capabilities = ["update"]
}

path "transit/keys/hr-audit-log-sig" {
    capabilities = ["read"]
}

path "transit/sign/hr-audit-log-sig" {
    capabilities = ["update"]
}

path "transit/verify/hr-audit-log-sig" {
    capabilities = ["update"]
}
