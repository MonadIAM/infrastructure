# infra

Shared infrastructure for MonadIAM services: Apache Kafka for inter-service messaging, HashiCorp Vault for shared secrets/crypto, ClickHouse as the WORM-style archive store for audit/change logs, and Grafana + Prometheus + Loki for monitoring. Deployed once per environment and shared by every service repo through three external Docker networks.

----

<details>
<summary><strong>Local Deployment</strong></summary>

Make sure Docker and Docker Compose V2 are installed on your system before proceeding.

1. Create shared Docker networks:
```sh
make network name="monitoring-network"
make network name="kafka-net"
make network name="vault-net"
```
2. Generate `.env` from example:
```sh
make env
```
3. Generate `.vault.env` from example (seed secrets for this repo's own `vault-init` — see the ClickHouse section below) and fill in real values:
```sh
cp .vault.env.example .vault.env
```
4. Start all services in detached mode:
```sh
make up
```
5. Verify Kafka is running:
```sh
make logs
```

</details>

----

<details>
<summary><strong>Networks</strong></summary>

Three external networks are created once and shared by every consuming service - this repo attaches to them, it doesn't own their lifecycle.

| Network              | Env var                     | Default              | Used by                                                 |
|:---------------------|:----------------------------|:---------------------|:--------------------------------------------------------|
| `kafka-net`          | `DOCKER_NETWORK_KAFKA`      | `kafka-net`          | Kafka, Vault init containers, service app               |
| `monitoring-network` | `DOCKER_NETWORK_MONITORING` | `monitoring-network` | Grafana, Prometheus, Loki, service app                  |
| `vault-net`          | `DOCKER_NETWORK_VAULT`      | `vault-net`          | Vault, each service's `vault-init`/`vault-secrets-init` |

**Connecting a consuming service:**
Add the ones it needs to that service's own `docker-compose.yml`:
```yaml
networks:
  kafka_net:
    external: true
    name: ${DOCKER_NETWORK_KAFKA:-kafka-net}
  vault_net:
    external: true
    name: ${DOCKER_NETWORK_VAULT:-vault-net}
  monitoring_net:
    external: true
    name: ${DOCKER_NETWORK_MONITORING:-monitoring-network}
```

</details>

----

<details>
<summary><strong>Kafka</strong></summary>

The broker is configured with **Advertised Listeners**, allowing two connection modes:

**External connection *(Host Machine)***
If your service runs locally via `npm run`:
- *Broker address:* `localhost:9092`
- *Protocol:* `PLAINTEXT`

**Internal connection *(Docker Network)***
If your service runs inside its own Docker container, on `kafka-net`:
- *Broker address:* `kafka:29092`
- *Protocol:* `PLAINTEXT`

</details>

----

<details>
<summary><strong>Vault</strong></summary>

A single shared Vault instance is run from this repository and used by all services, on `vault-net` - each service still owns its own secrets engines, KV paths, and policies, set up by that service's own `vault-init`/`vault-secrets-init` containers against this shared instance. This repo is also a consumer of its own shared Vault: `infra`'s own `vault-init`/`vault-secrets-init` (`infrastructure/vault/`) seed and render the ClickHouse passwords (see ClickHouse section) — same pattern as every other service, not a special case.

- *Address (Docker network):* `vault:8200`
- *Address (host machine):* `http://localhost:8200`
- *Root token (dev only):* value of `VAULT_ROOT_TOKEN` in this repo's `.env` - every consuming service's own `.env` must use the same value.

Dev mode is in-memory: all secrets/mounts are lost on restart, which is why every service's `vault-init` re-creates its own mounts/keys/policies idempotently on every startup.

</details>

----

<details>
<summary><strong>ClickHouse</strong></summary>

Single shared ClickHouse instance, run from this repository, on `kafka-net` (consumes archive events from Kafka) and `monitoring-network` (metrics scraping). Intended as the append-only long-term store for `AuditLog`/`ChangeLog` records replicated from each service's Postgres (`docs/BACKLOG.md`, п.14).

Schema (`audit.audit_log`, `audit.change_log`) is one shared table per entity across all services, not one per service — applied idempotently on every startup by `clickhouse-init` from `infrastructure/clickhouse/schema/*.sql`, the same pattern as `kafka-init`/`init-topics.sh`. Both tables carry a `service` discriminator column and a 365-day `TTL`. Kafka-engine ingestion tables (`docs/BACKLOG.md` п.14) are set up in a later step, not yet present at this stage.

- *Address (Docker network):* `clickhouse:9000` (native), `clickhouse:8123` (HTTP)
- *Address (host machine):* `localhost:9000` (native), `http://localhost:8123` (HTTP), `http://localhost:8123/play` (built-in query UI)
- *Default database:* `CLICKHOUSE_DB` in this repo's `.env`.

Three identities, not one shared login:

| User                     | Grants                      | Purpose                       |
|:-------------------------|:----------------------------|:------------------------------|
| `CLICKHOUSE_USER`        | admin, `access_management`  | schema DDL, break-glass only  |
| `CLICKHOUSE_WRITER_USER` | `INSERT`, `SELECT`          | Kafka-engine ingestion        |
| `CLICKHOUSE_READER_USER` | `SELECT`                    | Grafana / ad-hoc queries      |

Grants are on `audit.audit_log`/`audit.change_log`. Admin is for `clickhouse-init`/operator use only, not app or query traffic. Neither `audit_writer` nor `audit_reader` was ever granted `ALTER`/`DROP`/`DELETE`/`UPDATE` — ClickHouse's access control is default-deny, so this isn't a revoke, it's simply never having granted it. This is what actually enforces WORM: the storage engine itself doesn't prevent mutations, permissions do.

All three passwords come from Vault, not from `.env`: seeded once from `.vault.env` (gitignored, see `.vault.env.example`) by `vault-init`, then fetched and written to `/vault/secrets/*` files by `vault-secrets-init` — `clickhouse` and `clickhouse-init` read them from there (`docker/postgres/entrypoint.sh` in `identity-service` is the same pattern this mirrors). Usernames aren't secrets and stay plain in `.env`.

</details>

----

<details>
<summary><strong>Commands</strong></summary>

| Command                                       | Makefile                | Description                                                                  |
|:----------------------------------------------|:------------------------|:-----------------------------------------------------------------------------|
| `docker compose up -d`                        | `make up`               | Start all services in detached mode.                                         |
| `docker compose stop`                         | `make stop`             | Stop containers without removing them.                                       |
| `docker compose down`                         | `make down`             | Stop and remove containers and networks.                                     |
| `docker compose down && docker-compose up -d` | `make restart`          | Restart the entire infrastructure.                                           |
| `docker compose logs -f kafka`                | `make logs`             | Stream Kafka logs in real time.                                              |
| `docker exec -it kafka /bin/bash`             | `make exec`             | Open a shell inside the Kafka container.                                     |
| `docker compose down -v`                      | `make clean`            | Remove containers along with volumes.                                        |
| `docker system prune -f`                      | `make prune`            | Global cleanup of unused Docker resources.                                   |
| `docker network create --name=$(name)`        | `make network name="*"` | Create a new docker network.                                                 |
| `docker network ls`                           | `make networks`         | List networks (verify `kafka-net`, `monitoring-network`, `vault-net` exist). |

</details>

----

<details>
<summary><strong>UI Links</strong></summary>

| Service            | URL                        |
|:-------------------|:---------------------------|
| Grafana            | http://localhost:3000      |
| Kafka UI           | http://localhost:8080      |
| Vault UI           | http://localhost:8200/ui   |
| ClickHouse Play UI | http://localhost:8123/play |

</details>

----
