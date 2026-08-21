-include .env

IDENTITY_POSTGRES_REPLICAS ?= 2
ACCESS_CONTROL_POSTGRES_REPLICAS ?= 2
NOTIFICATION_POSTGRES_REPLICAS ?= 2

COMPOSE_FILES = -f docker-compose.yml \
	-f docker-compose.identity.yml \
	-f docker-compose.access-control.yml \
	-f docker-compose.notification.yml \
	-f docker-compose.routing.yml \
	-f docker-compose.kafka.yml \
	-f docker-compose.observability.yml \
	-f docker-compose.audit-archive.yml
COMPOSE = docker compose $(COMPOSE_FILES)
POSTGRES_SCALE = --scale identity-postgresql=$(IDENTITY_POSTGRES_REPLICAS) \
	--scale access-control-postgresql=$(ACCESS_CONTROL_POSTGRES_REPLICAS) \
	--scale notification-postgresql=$(NOTIFICATION_POSTGRES_REPLICAS)
TERRAFORM_RUNTIME_STATE = infrastructure/terraform/terraform.tfstate \
	infrastructure/terraform/terraform.tfstate.backup \
	infrastructure/terraform/.terraform.tfstate.lock.info \
	infrastructure/terraform/.output

env:
	[ -f .env.example ] && cp -f .env.example .env || echo '.env.example not found'
build-consul-envoy:
	$(COMPOSE) build consul-envoy-image
build-postgres:
	$(COMPOSE) build identity-postgresql access-control-postgresql notification-postgresql
build-routing:
	$(COMPOSE) build postgres-haproxy pgbouncer-write pgbouncer-read
up: build-consul-envoy build-postgres build-routing
	$(COMPOSE) up -d $(POSTGRES_SCALE)
stop:
	$(COMPOSE) stop
down:
	$(COMPOSE) down
config:
	$(COMPOSE) config
clean:
	$(COMPOSE) down -v
	rm -rf infrastructure/vault/tls infrastructure/vault/init-keys
	rm -rf $(TERRAFORM_RUNTIME_STATE)
	echo "Docker volumes, Vault TLS material, Vault init/unseal keys, and Terraform runtime state removed - next 'make up' will re-init Vault from scratch."
restart: down up
logs:
	$(COMPOSE) logs -f kafka
network:
	docker network create $(name)
networks:
	docker network ls
exec:
	docker exec -it kafka /bin/bash
provisioner-token:
	cat infrastructure/terraform/.output/provisioner-token-$(service) | pbcopy
	echo "VAULT_PROVISIONER_TOKEN for $(service) copied to clipboard"
vault-reset:
	$(COMPOSE) stop vault vault-init-unseal vault-tls service-tls
	$(COMPOSE) rm -f vault vault-init-unseal vault-tls service-tls
	docker volume rm -f infra_vault_data
	docker volume rm -f $${DOCKER_VOLUME_VAULT_CA:?}
	docker volume rm -f $${DOCKER_VOLUME_SERVICE_TLS:?}
	docker volume rm -f service_tls_ca
	rm -rf infrastructure/vault/tls infrastructure/vault/init-keys
	rm -rf $(TERRAFORM_RUNTIME_STATE)
	echo "Vault/service TLS material, init/unseal keys, Raft storage, and Terraform runtime state removed - next 'make up' will re-init from scratch."
pg-reset:
	@[ -n "$(service)" ] || (echo "Usage: make pg-reset service=identity|access-control|notification" >&2; exit 1)
	@case "$(service)" in \
		identity|identity-service) svc=identity-postgresql ;; \
		access-control|access-control-service) svc=access-control-postgresql ;; \
		notification|notification-service) svc=notification-postgresql ;; \
		*) echo "Unknown service: $(service)" >&2; exit 1 ;; \
	esac; \
	$(COMPOSE) stop $$svc; \
	$(COMPOSE) rm -f -v $$svc
pg-reset-all:
	$(COMPOSE) stop identity-postgresql access-control-postgresql notification-postgresql
	$(COMPOSE) rm -f -v identity-postgresql access-control-postgresql notification-postgresql
pg-clean-orphans:
	$(COMPOSE) down --remove-orphans
pg-list:
	@[ -n "$(service)" ] || (echo "Usage: make pg-list service=identity|access-control|notification" >&2; exit 1)
	@case "$(service)" in \
		identity|identity-service) svc=identity-postgresql ;; \
		access-control|access-control-service) svc=access-control-postgresql ;; \
		notification|notification-service) svc=notification-postgresql ;; \
		*) echo "Unknown service: $(service)" >&2; exit 1 ;; \
	esac; \
	$(COMPOSE) exec $$svc patronictl -c /tmp/patroni.yml list
