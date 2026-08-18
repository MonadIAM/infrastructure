env:
	[ -f .env.example ] && cp -f .env.example .env || echo '.env.example not found'
up:
	docker compose up -d
stop:
	docker compose stop
down:
	docker compose down
clean:
	docker compose down -v
	rm -rf infrastructure/vault/tls infrastructure/vault/init-keys
	echo "Docker volumes, Vault TLS material, and Vault init/unseal keys removed - next 'make up' will re-init Vault from scratch."
restart: down up
logs:
	docker compose logs -f kafka
network:
	docker network create $(name)
networks:
	docker network ls
exec:
	docker exec -it kafka /bin/bash
provisioner-token:
	cat terraform/.output/provisioner-token-$(service) | pbcopy
	echo "VAULT_PROVISIONER_TOKEN for $(service) copied to clipboard"
vault-reset:
	docker compose stop vault vault-init-unseal vault-tls
	docker compose rm -f vault vault-init-unseal vault-tls
	docker volume rm -f infra_vault_data
	docker volume rm -f $${DOCKER_VOLUME_VAULT_CA:-vault_ca}
	rm -rf infrastructure/vault/tls infrastructure/vault/init-keys
	echo "Vault TLS material, init/unseal keys, and Raft storage removed - next 'make up' will re-init from scratch."
