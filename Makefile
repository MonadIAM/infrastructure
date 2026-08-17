env:
	[ -f .env.example ] && cp -f .env.example .env || echo '.env.example not found'
up:
	docker compose up -d
stop:
	docker compose stop
down:
	docker compose down
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
