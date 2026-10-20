path "kv/data/monadiam-data/postgresql/organization/app" {
    capabilities = ["read"]
}

path "kv/data/monadiam-data/redis/organization/auth" {
    capabilities = ["read"]
}

path "transit/hmac/organization-audit-mask-hmac" {
    capabilities = ["update"]
}

path "transit/keys/organization-audit-log-sig" {
    capabilities = ["read"]
}

path "transit/sign/organization-audit-log-sig" {
    capabilities = ["update"]
}

path "transit/verify/organization-audit-log-sig" {
    capabilities = ["update"]
}
