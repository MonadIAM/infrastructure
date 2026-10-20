path "kv/data/monadiam-data/postgresql/notification/app" {
    capabilities = ["read"]
}

path "kv/data/monadiam-data/redis/notification/auth" {
    capabilities = ["read"]
}

path "transit/hmac/notification-audit-mask-hmac" {
    capabilities = ["update"]
}

path "transit/keys/notification-audit-log-sig" {
    capabilities = ["read"]
}

path "transit/sign/notification-audit-log-sig" {
    capabilities = ["update"]
}

path "transit/verify/notification-audit-log-sig" {
    capabilities = ["update"]
}
