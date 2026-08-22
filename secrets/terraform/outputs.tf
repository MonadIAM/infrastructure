output "identity_service_provisioner_token" {
    value     = vault_token.provisioner["identity-service"].client_token
    sensitive = true
}

output "access_control_service_provisioner_token" {
    value     = vault_token.provisioner["access-control-service"].client_token
    sensitive = true
}

output "notification_service_provisioner_token" {
    value     = vault_token.provisioner["notification-service"].client_token
    sensitive = true
}

output "consul_connect_ca_token" {
    value     = vault_token.consul_connect_ca.client_token
    sensitive = true
}

output "service_tls_issuer_token" {
    value     = vault_token.service_tls_issuer.client_token
    sensitive = true
}


output "infra_agent_bootstrap_token" {
    value     = vault_token.infra_agent_bootstrap.client_token
    sensitive = true
}

output "service_tls_ca_bundle" {
    value = format("%s\n%s", vault_pki_secret_backend_root_sign_intermediate.service_tls.certificate, vault_pki_secret_backend_root_cert.root.certificate)
}
