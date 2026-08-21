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

resource "vault_mount" "service_pki" {
    path                      = local.service_pki_mount
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

resource "vault_pki_secret_backend_issuer" "root" {
    backend     = vault_mount.pki_root.path
    issuer_ref  = vault_pki_secret_backend_root_cert.root.issuer_id
    issuer_name = "${var.environment}-root"
}

resource "vault_pki_secret_backend_config_issuers" "pki_root" {
    backend                       = vault_mount.pki_root.path
    default                       = vault_pki_secret_backend_issuer.root.issuer_id
    default_follows_latest_issuer = false
}

resource "vault_pki_secret_backend_intermediate_cert_request" "grpc" {
    depends_on = [vault_mount.pki]

    backend            = vault_mount.pki.path
    type               = "internal"
    common_name        = "MonadIAM Local Connect Intermediate CA"
    key_name           = "${var.environment}-connect-intermediate-key"
    key_type           = "rsa"
    key_bits           = 4096
    private_key_format = "der"
}

resource "vault_pki_secret_backend_root_sign_intermediate" "grpc" {
    depends_on = [vault_pki_secret_backend_config_issuers.pki_root]

    backend     = vault_mount.pki_root.path
    csr         = vault_pki_secret_backend_intermediate_cert_request.grpc.csr
    common_name = "MonadIAM Local Connect Intermediate CA"
    ttl         = 157680000
    format      = "pem"
    issuer_ref  = vault_pki_secret_backend_issuer.root.issuer_id
}

resource "vault_pki_secret_backend_intermediate_set_signed" "grpc" {
    backend     = vault_mount.pki.path
    certificate = vault_pki_secret_backend_root_sign_intermediate.grpc.certificate
}

resource "vault_pki_secret_backend_issuer" "grpc" {
    backend     = vault_mount.pki.path
    issuer_ref  = vault_pki_secret_backend_intermediate_set_signed.grpc.imported_issuers[0]
    issuer_name = "${var.environment}-connect-intermediate"
}

resource "vault_pki_secret_backend_config_issuers" "grpc" {
    backend                       = vault_mount.pki.path
    default                       = vault_pki_secret_backend_issuer.grpc.issuer_id
    default_follows_latest_issuer = false
}

resource "vault_pki_secret_backend_intermediate_cert_request" "service_tls" {
    depends_on = [vault_mount.service_pki]

    backend            = vault_mount.service_pki.path
    type               = "internal"
    common_name        = "MonadIAM Local Service TLS Intermediate CA"
    key_name           = "${var.environment}-service-tls-intermediate-key"
    key_type           = "rsa"
    key_bits           = 4096
    private_key_format = "der"
}

resource "vault_pki_secret_backend_root_sign_intermediate" "service_tls" {
    depends_on = [vault_pki_secret_backend_config_issuers.pki_root]

    backend     = vault_mount.pki_root.path
    csr         = vault_pki_secret_backend_intermediate_cert_request.service_tls.csr
    common_name = "MonadIAM Local Service TLS Intermediate CA"
    ttl         = 157680000
    format      = "pem"
    issuer_ref  = vault_pki_secret_backend_issuer.root.issuer_id
}

resource "vault_pki_secret_backend_intermediate_set_signed" "service_tls" {
    backend     = vault_mount.service_pki.path
    certificate = vault_pki_secret_backend_root_sign_intermediate.service_tls.certificate
}

resource "vault_pki_secret_backend_issuer" "service_tls" {
    backend     = vault_mount.service_pki.path
    issuer_ref  = vault_pki_secret_backend_intermediate_set_signed.service_tls.imported_issuers[0]
    issuer_name = "${var.environment}-service-tls-intermediate"
}

resource "vault_pki_secret_backend_config_issuers" "service_tls" {
    backend                       = vault_mount.service_pki.path
    default                       = vault_pki_secret_backend_issuer.service_tls.issuer_id
    default_follows_latest_issuer = false
}

resource "vault_pki_secret_backend_role" "pgbouncer_server" {
    backend = vault_mount.service_pki.path
    name    = "pgbouncer-server"

    ttl        = 31536000
    max_ttl    = 31536000
    issuer_ref = vault_pki_secret_backend_issuer.service_tls.issuer_id

    allowed_domains    = ["pgbouncer", "pgbouncer-write", "pgbouncer-read", "localhost"]
    allow_bare_domains = true
    allow_ip_sans      = true
    require_cn         = true

    server_flag = true
    client_flag = false

    key_type = "rsa"
    key_bits = 2048
}

resource "vault_pki_secret_backend_role" "postgres_server" {
    backend = vault_mount.service_pki.path
    name    = "postgres-server"

    ttl        = 31536000
    max_ttl    = 31536000
    issuer_ref = vault_pki_secret_backend_issuer.service_tls.issuer_id

    allowed_domains = [
        "postgres",
        "postgres-haproxy",
        "identity-postgresql",
        "access-control-postgresql",
        "notification-postgresql",
        "localhost",
    ]
    allow_bare_domains = true
    allow_ip_sans      = true
    require_cn         = true

    server_flag = true
    client_flag = false

    key_type = "rsa"
    key_bits = 2048
}

resource "vault_pki_secret_backend_role" "redis_server" {
    backend = vault_mount.service_pki.path
    name    = "redis-server"

    ttl        = 31536000
    max_ttl    = 31536000
    issuer_ref = vault_pki_secret_backend_issuer.service_tls.issuer_id

    allowed_domains = [
        "redis",
        "identity-redis",
        "access-control-redis",
        "notification-redis",
        "template-redis",
        "localhost",
    ]
    allow_bare_domains = true
    allow_ip_sans      = true
    require_cn         = true

    server_flag = true
    client_flag = false

    key_type = "rsa"
    key_bits = 2048
}

resource "vault_pki_secret_backend_role" "app_client" {
    backend = vault_mount.service_pki.path
    name    = "app-client"

    ttl        = 31536000
    max_ttl    = 31536000
    issuer_ref = vault_pki_secret_backend_issuer.service_tls.issuer_id

    allowed_domains    = ["monadiam-app-client"]
    allow_bare_domains = true
    allow_ip_sans      = false
    require_cn         = true

    server_flag = false
    client_flag = true

    key_type = "rsa"
    key_bits = 2048
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

resource "vault_policy" "service_tls_issuer" {
    name = "service-tls-issuer"

    policy = <<-EOT
        path "${local.service_pki_mount}/issue/pgbouncer-server" {
          capabilities = ["update"]
        }

        path "${local.service_pki_mount}/issue/postgres-server" {
          capabilities = ["update"]
        }

        path "${local.service_pki_mount}/issue/redis-server" {
          capabilities = ["update"]
        }

        path "${local.service_pki_mount}/issue/app-client" {
          capabilities = ["update"]
        }

        path "auth/token/renew-self" {
          capabilities = ["update"]
        }

        path "auth/token/lookup-self" {
          capabilities = ["read"]
        }
    EOT
}

resource "vault_token" "service_tls_issuer" {
    policies          = [vault_policy.service_tls_issuer.name]
    no_default_policy = true
    renewable         = true
    ttl               = "720h"
}
