variable "notification_aws_region" {
    type    = string
    default = "eu-central-1"
}

variable "notification_aws_root_access_key_id" {
    type      = string
    default   = ""
    sensitive = true
}

variable "notification_aws_root_secret_access_key" {
    type      = string
    default   = ""
    sensitive = true
}

resource "vault_aws_secret_backend" "notification" {
    path       = local.aws_mount
    region     = var.notification_aws_region
    access_key = var.notification_aws_root_access_key_id
    secret_key = var.notification_aws_root_secret_access_key

    default_lease_ttl_seconds = 3600
    max_lease_ttl_seconds     = 86400
}

resource "vault_aws_secret_backend_role" "notification_ses" {
    backend         = vault_aws_secret_backend.notification.path
    name            = "notification-service-ses"
    credential_type = "iam_user"

    policy_document = jsonencode({
        Version = "2012-10-17"
        Statement = [
            {
                Effect = "Allow"
                Action = [
                    "ses:SendEmail",
                    "ses:SendRawEmail",
                    "sns:Publish",
                ]
                Resource = "*"
            },
        ]
    })
}
