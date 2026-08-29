env:
	[ -f .env.example ] && cp -f .env.example .env || echo '.env.example not found'

up:
	docker compose up -d

down:
	docker compose down

restart: down up

stop:
	docker compose stop

start:
	docker compose start
