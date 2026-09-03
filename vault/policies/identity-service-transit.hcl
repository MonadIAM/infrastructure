path "transit/keys/identity-access-token" {
  capabilities = ["read"]
}

path "transit/sign/identity-access-token" {
  capabilities = ["update"]
}

path "transit/verify/identity-access-token" {
  capabilities = ["update"]
}

path "transit/keys/identity-refresh-token" {
  capabilities = ["read"]
}

path "transit/sign/identity-refresh-token" {
  capabilities = ["update"]
}

path "transit/verify/identity-refresh-token" {
  capabilities = ["update"]
}

path "transit/keys/identity-id-token" {
  capabilities = ["read"]
}

path "transit/sign/identity-id-token" {
  capabilities = ["update"]
}

path "transit/verify/identity-id-token" {
  capabilities = ["update"]
}

path "transit/keys/identity-totp-secret" {
  capabilities = ["read"]
}

path "transit/encrypt/identity-totp-secret" {
  capabilities = ["update"]
}

path "transit/decrypt/identity-totp-secret" {
  capabilities = ["update"]
}

path "transit/rewrap/identity-totp-secret" {
  capabilities = ["update"]
}

path "transit/hmac/identity-pairwise-sub" {
  capabilities = ["update"]
}

path "transit/hmac/identity-audit-mask-hmac" {
  capabilities = ["update"]
}

path "transit/keys/identity-audit-log-sig" {
  capabilities = ["read"]
}

path "transit/sign/identity-audit-log-sig" {
  capabilities = ["update"]
}

path "transit/verify/identity-audit-log-sig" {
  capabilities = ["update"]
}
