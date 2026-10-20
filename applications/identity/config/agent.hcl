pid_file = "/tmp/vault-agent.pid"

vault {
    address = "https://vault.vault.svc.cluster.local:8200"
    ca_cert = "/tls/vault/ca.crt"
}

auto_auth {
    method "kubernetes" {
        mount_path = "auth/kubernetes"
        config = {
            role = "identity-service"
            token_path = "/var/run/secrets/vault/token"
        }
    }
}

cache {
    use_auto_auth_token = true
}

listener "unix" {
    address = "/vault/proxy/agent.sock"
    socket_mode = "0660"
    tls_disable = true
}
