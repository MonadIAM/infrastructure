locals {
    kv_mount       = "kv"
    infra_kv_mount = "infra-kv"
    database_mount = "database"
    transit_mount  = "transit"
    aws_mount      = "aws"

    services = {
        identity-service = {
            postgres_host       = "identity-service-postgres-primary"
            postgres_port       = 5432
            postgres_db         = "identity-service"
            postgres_admin_user = "user"

            kv_runtime_path     = "identity-service/runtime"
            kv_replication_path = "identity-service/postgresql-replication"
            kv_oauth_path       = "identity-service/oauth"

            has_transit  = true
            has_oauth    = true
            has_aws      = false
            has_key_trim = true

            kubernetes_namespace       = "identity-service"
            kubernetes_service_account = "identity-service"
        }

        access-control-service = {
            postgres_host       = "access-control-service-postgres-primary"
            postgres_port       = 5432
            postgres_db         = "access-control-service"
            postgres_admin_user = "user"

            kv_runtime_path     = "access-control-service/runtime"
            kv_replication_path = "access-control-service/postgresql-replication"
            kv_oauth_path       = null

            has_transit  = false
            has_oauth    = false
            has_aws      = false
            has_key_trim = false

            kubernetes_namespace       = "access-control-service"
            kubernetes_service_account = "access-control-service"
        }

        notification-service = {
            postgres_host       = "notification-service-postgres-primary"
            postgres_port       = 5432
            postgres_db         = "notification-service"
            postgres_admin_user = "user"

            kv_runtime_path     = "notification-service/runtime"
            kv_replication_path = "notification-service/postgresql-replication"
            kv_oauth_path       = null

            has_transit  = false
            has_oauth    = false
            has_aws      = true
            has_key_trim = false

            kubernetes_namespace       = "notification-service"
            kubernetes_service_account = "notification-service"
        }
    }
}
