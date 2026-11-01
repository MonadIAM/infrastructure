terraform {
  required_version = "= 1.17.1"
  backend "local" {}
  required_providers {
    docker = {
      source  = "kreuzwerker/docker"
      version = "4.6.0"
    }
  }
}
variable "docker_host" {
  type = string
  validation {
    condition     = trimspace(var.docker_host) != ""
    error_message = "docker_host is required."
  }
}
variable "image" { type = string }
variable "port" { type = number }
variable "network" { type = string }
variable "project" { type = string }
variable "config_path" { type = string }
variable "container_name" {
  type = string
}
provider "docker" { host = var.docker_host }

resource "docker_network" "vault" {
  name = var.network
  lifecycle {
    prevent_destroy = true
    ignore_changes  = [labels]
  }
}
resource "docker_volume" "data" {
  name = "${var.project}_vault_data"
  lifecycle {
    prevent_destroy = true
    ignore_changes  = [labels]
  }
}
resource "docker_volume" "init" {
  name = "${var.project}_vault_init"
  lifecycle {
    prevent_destroy = true
    ignore_changes  = [labels]
  }
}
resource "docker_image" "vault" {
  name         = var.image
  keep_locally = true
}
resource "docker_container" "vault" {
  name       = var.container_name
  image      = docker_image.vault.image_id
  restart    = "on-failure"
  entrypoint = ["vault"]
  command    = ["server", "-config=/vault/config/vault.hcl"]
  capabilities { add = ["CAP_IPC_LOCK"] }
  ports {
    internal = 8200
    external = var.port
  }
  volumes {
    host_path      = var.config_path
    container_path = "/vault/config/vault.hcl"
    read_only      = true
  }
  volumes {
    volume_name    = docker_volume.data.name
    container_path = "/vault/file"
  }
  networks_advanced {
    name    = docker_network.vault.name
    aliases = ["vault"]
  }
  healthcheck {
    test         = ["CMD-SHELL", "VAULT_ADDR=http://127.0.0.1:8200 vault status >/dev/null 2>&1; code=$?; [ $code -eq 0 ] || [ $code -eq 2 ]"]
    interval     = "5s"
    timeout      = "3s"
    retries      = 40
    start_period = "5s"
  }
  wait         = true
  wait_timeout = 210
}
