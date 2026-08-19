resource "vault_mount" "pki_root" {
    path                      = local.pki_root_mount
    type                      = "pki"
    default_lease_ttl_seconds = 86400
    max_lease_ttl_seconds     = 315360000
}

resource "vault_mount" "pki" {
    path                      = local.pki_mount
    type                      = "pki"
    default_lease_ttl_seconds = 86400
    max_lease_ttl_seconds     = 157680000
}

resource "vault_pki_secret_backend_root_cert" "root" {
    depends_on = [vault_mount.pki_root]

    backend              = vault_mount.pki_root.path
    type                 = "internal"
    common_name          = "MonadIAM Local Root CA"
    ttl                  = 315360000
    format               = "pem"
    private_key_format   = "der"
    key_type             = "rsa"
    key_bits             = 4096
    exclude_cn_from_sans = true
}

resource "vault_pki_secret_backend_intermediate_cert_request" "grpc" {
    depends_on = [vault_mount.pki]

    backend            = vault_mount.pki.path
    type               = "internal"
    common_name        = "MonadIAM Local Connect Intermediate CA"
    key_type           = "rsa"
    key_bits           = 4096
    private_key_format = "der"
}

resource "vault_pki_secret_backend_root_sign_intermediate" "grpc" {
    depends_on = [vault_pki_secret_backend_root_cert.root]

    backend     = vault_mount.pki_root.path
    csr         = vault_pki_secret_backend_intermediate_cert_request.grpc.csr
    common_name = "MonadIAM Local Connect Intermediate CA"
    ttl         = 157680000
    format      = "pem_bundle"
    issuer_ref  = vault_pki_secret_backend_root_cert.root.issuer_id
}

resource "vault_pki_secret_backend_intermediate_set_signed" "grpc" {
    backend     = vault_mount.pki.path
    certificate = vault_pki_secret_backend_root_sign_intermediate.grpc.certificate
}

resource "vault_policy" "consul_connect_ca" {
    name = "consul-connect-ca"

    policy = <<-EOT
        path "sys/mounts/${local.pki_root_mount}" {
          capabilities = ["read"]
        }

        path "sys/mounts/${local.pki_mount}" {
          capabilities = ["read"]
        }

        path "sys/mounts/${local.pki_mount}/tune" {
          capabilities = ["update"]
        }

        path "${local.pki_root_mount}/" {
          capabilities = ["read"]
        }

        path "${local.pki_root_mount}/root/sign-intermediate" {
          capabilities = ["update"]
        }

        path "${local.pki_mount}/*" {
          capabilities = ["create", "read", "update", "delete", "list"]
        }

        path "auth/token/renew-self" {
          capabilities = ["update"]
        }

        path "auth/token/lookup-self" {
          capabilities = ["read"]
        }
    EOT
}

resource "vault_token" "consul_connect_ca" {
    policies          = [vault_policy.consul_connect_ca.name]
    no_default_policy = true
    renewable         = true
    ttl               = "720h"
}
