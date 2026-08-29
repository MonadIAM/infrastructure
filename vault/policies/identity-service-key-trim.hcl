path "transit/keys/identity-access-token" {
  capabilities = ["read", "update"]
}

path "transit/keys/identity-access-token/config" {
  capabilities = ["update"]
}

path "transit/keys/identity-access-token/trim" {
  capabilities = ["update"]
}

path "transit/keys/identity-refresh-token" {
  capabilities = ["read", "update"]
}

path "transit/keys/identity-refresh-token/config" {
  capabilities = ["update"]
}

path "transit/keys/identity-refresh-token/trim" {
  capabilities = ["update"]
}

path "transit/keys/identity-id-token" {
  capabilities = ["read", "update"]
}

path "transit/keys/identity-id-token/config" {
  capabilities = ["update"]
}

path "transit/keys/identity-id-token/trim" {
  capabilities = ["update"]
}
