path "kv/data/monadiam-data/postgresql/access-control/app" {
    capabilities = ["read"]
}

path "kv/data/monadiam-data/redis/access-control/auth" {
    capabilities = ["read"]
}

path "transit/hmac/access-control-audit-mask-hmac" {
    capabilities = ["update"]
}

path "transit/keys/access-control-audit-log-sig" {
    capabilities = ["read"]
}

path "transit/sign/access-control-audit-log-sig" {
    capabilities = ["update"]
}

path "transit/verify/access-control-audit-log-sig" {
    capabilities = ["update"]
}
