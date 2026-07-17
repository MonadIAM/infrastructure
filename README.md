# @monadiam/infra

Shared infrastructure for MonadIAM services: Apache Kafka for inter-service messaging, HashiCorp Vault for shared secrets/crypto, ClickHouse as the WORM-style mid-term archive store for audit/change logs, MinIO as the COLD long-term archive those logs age out into, and Grafana + Prometheus + Loki for monitoring. Deployed once per environment and shared by every service repo through three external Docker networks.

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
2. Generate `.env` and `.vault.env` from example (the latter seeds secrets for this repo's own `vault-init` — see the ClickHouse/MinIO sections below) and fill in real values:
```sh
make env
```
3. Start all services in detached mode:
```sh
make up
```
4. Verify Kafka is running:
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

Single shared ClickHouse instance, run from this repository, on `kafka-net` (Kafka ingestion) and `monitoring-network` (metrics). Append-only long-term store for `AuditLog`/`ChangeLog`, replicated from each service's Postgres.

- *Address (Docker network):* `clickhouse:9000` (native), `clickhouse:8123` (HTTP)
- *Address (host machine):* `localhost:9000` (native), `http://localhost:8123` (HTTP), `http://localhost:8123/play`
- *Default database:* `CLICKHOUSE_DB` in `.env`

Schema applied in filename order by `clickhouse-init`, from `infrastructure/clickhouse/schema/*.sql`:

| File        | What                                                                                  |
|:------------|:--------------------------------------------------------------------------------------|
| `001`/`002` | `audit.audit_log`/`audit.change_log` — shared tables, `service` column, 180-day `TTL` |
| `003`       | `audit_writer`/`audit_reader` users + grants                                          |
| `004`       | Kafka-engine queue tables (`audit-log-archive`/`change-log-archive`)                  |
| `005`       | Materialized views, `DEFINER = audit_writer`                                          |
| `006`       | `audit.archive_log` — bookkeeping table for `clickhouse-archiver` (see MinIO section) |

`003`–`006` use `{{writer_user}}`-style placeholders, substituted via `sed` before running — same convention as the Vault HCL policies.

TTL is 180 days, not the retention period itself — ClickHouse is the mid-term investigation surface (WORM via RBAC below), not the final resting place. `clickhouse-archiver` copies each partition into MinIO well before the TTL would reclaim it (see MinIO section); the TTL is a backstop, not the primary deletion path in normal operation.

Producers write to their own `system.outbox` table (transactional outbox pattern) — Debezium's `EventRouter` wraps every row as `{ actionType, payload: {...} }` before it reaches Kafka, same convention as `realm`/`membership`/`blacklist`. `004`'s queue tables ingest that envelope as-is (`actionType String, payload String`); `005`'s MV unwraps `payload` into the real columns via `JSONExtractString`. Producer-side field names inside `payload` match the columns in `001`/`002` exactly (snake_case); `input`/`delta` arrive pre-serialized as a JSON string, not a nested object.

Three identities, not one shared login:

| User                     | Grants              | Purpose                      |
|:-------------------------|:--------------------|:-----------------------------|
| `CLICKHOUSE_USER`        | admin               | schema DDL, break-glass only |
| `CLICKHOUSE_WRITER_USER` | `INSERT`, `SELECT`  | Kafka ingestion              |
| `CLICKHOUSE_READER_USER` | `SELECT`            | Grafana / ad-hoc queries     |

No `ALTER`/`DROP`/`DELETE`/`UPDATE` for writer/reader — never granted, not revoked. That's what enforces WORM, not the storage engine.

Passwords come from Vault, not `.env` (see Vault section) — mounted into `clickhouse`/`clickhouse-init` at `/vault/secrets/*`. Usernames stay plain in `.env`.

</details>

----

<details>
<summary><strong>MinIO</strong></summary>

COLD archive for `audit.audit_log`/`audit.change_log` — where data ends up after it ages out of ClickHouse's 180-day hot window, as Parquet. Single shared MinIO instance, run from this repository, on `kafka-net`.

- *Address (Docker network):* `minio:9000` (S3 API), `minio:9090` (console)
- *Address (host machine):* `http://localhost:${MINIO_API_PORT}` (S3 API), `http://localhost:${MINIO_CONSOLE_PORT}` (console)
- *Buckets:* `MINIO_AUDIT_LOG_BUCKET` (default `audit-archive`), `MINIO_CHANGE_LOG_BUCKET` (default `change-log-archive`) — one per ClickHouse table, both services land in the same bucket (`service` column already discriminates inside the Parquet file, same as in ClickHouse)

Buckets are created by `minio-init` (`infrastructure/minio/init.sh`, `minio/mc` image) with Object Lock enabled at creation time — retrofitting Object Lock onto an existing bucket isn't possible, which is why bucket creation and lock configuration happen in the same script, once. Default retention is set in **COMPLIANCE** mode (not GOVERNANCE): nobody, including root, can shorten it or delete an object early. Retention length equals lifecycle expiry — objects age out of the bucket exactly when WORM protection itself lapses, not before:

| Bucket                    | Retention (`MINIO_*_RETENTION_DAYS`) |
|:---------------------------|:-------------------------------------|
| `audit-archive`            | 1095 days (3 years)                  |
| `change-log-archive`       | 365 days (1 year)                    |

`minio-init` also provisions a least-privilege `archiver` identity (`MINIO_ARCHIVER_ACCESS_KEY`, policy in `infrastructure/minio/archiver-policy.json`) scoped to `s3:PutObject`/`s3:GetObject` on just these two buckets — no delete, no list, no admin. This is the only credential `clickhouse-archiver` ever sees.

Root credentials (`MINIO_ROOT_USER`, password from Vault) exist solely to bootstrap buckets/policy/user in `minio-init` — nothing else uses them.

</details>

----

<details>
<summary><strong>clickhouse-archiver</strong></summary>

Daily job (`infrastructure/archiver/`, plain shell loop — no cron daemon, no NestJS: the job is platform-level, not owned by identity-service or access-control-service) that exports ClickHouse partitions into MinIO as Parquet once they're older than `ARCHIVE_AFTER_DAYS` (default 150 — a 30-day margin before the 180-day ClickHouse `TTL`), verifies the export by row count, and records success in `audit.archive_log`.

It never issues `DROP PARTITION` or any other destructive statement against ClickHouse — deletion stays exactly where it already was, the table `TTL`. If the archiver falls behind or breaks, the worst case is a few extra days of ClickHouse storage, not a race against data loss; the margin exists specifically so a stalled job has time to be noticed and fixed before the TTL would otherwise catch up.

Uses `CLICKHOUSE_IMAGE` unmodified (`clickhouse-client` only, no extra tooling) — export, read-back verification, and the eventual queries against archived Parquet from ClickHouse Play UI (`s3()` table function, works for both writing and reading MinIO) all go through the same one binary.

> `mc ilm`/`mc retention` flag names in `infrastructure/minio/init.sh` have shifted across `mc` releases — double-check them against the exact pinned `MINIO_MC_IMAGE` tag before relying on this in production.

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

| Service            | URL                          |
|:-------------------|:-----------------------------|
| Grafana            | http://localhost:3000        |
| Kafka UI           | http://localhost:8080        |
| Vault UI           | http://localhost:8200/ui     |
| ClickHouse Play UI | http://localhost:8123/play   |
| MinIO Console      | http://localhost:9011        |

</details>

----
