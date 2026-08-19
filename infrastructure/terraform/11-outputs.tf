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
