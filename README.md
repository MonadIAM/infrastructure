# infra

This repository contains the configuration for deploying shared Apache Kafka infrastructure. Designed to enable communication between independent microservices via a dedicated Docker network.

----

<details>
<summary><strong>Local Deployment</strong></summary>

Make sure Docker and Docker Compose V2 are installed on your system before proceeding.

1. Create shared Docker networks for inter-service communication and monitoring:
```sh
docker network create kafka-net
docker network create monitoring-network
```

2. Generate `.env` from example:
```sh
cp .env.example .env
```

3. Start all services in detached mode:
```sh
make up
```

4. Verify Kafka is running:
```sh
docker-compose logs -f kafka
```

</details>

----

<details>
<summary><strong>Kafka</strong></summary>

The infrastructure uses Apache Kafka in **KRaft mode** (no Zookeeper) with a single broker for local development.

#### Topics
| Topic       | Type       | Description                                     |
|:------------|:-----------|:------------------------------------------------|
| `assign`    | retention  | Role assignment events on account registration. |
| `blacklist` | compaction | Revoked/terminated session events.              |

#### Checking topic status
```sh
docker exec kafka /opt/kafka/bin/kafka-topics.sh \
  --bootstrap-server kafka:29092 \
  --describe
```

</details>

----

<details>
<summary><strong>Network</strong></summary>

The broker is configured with **Advertised Listeners**, allowing two connection modes:

#### External connection (Host Machine)
If your service runs locally via `npm run`:
- **Broker address:** `localhost:9092`
- **Protocol:** `PLAINTEXT`

#### Internal connection (Docker Network)
If your service runs inside its own Docker container:
- **Broker address:** `kafka:29092`
- **Protocol:** `PLAINTEXT`

#### Connecting external microservices:
Add the following to your service's `docker-compose.yml`:

```yaml
networks:
  kafka_net:
    external: true
    name: kafka-net
```

</details>

----

<details>
<summary><strong>Monitoring</strong></summary>

| Service               | URL                    | Description              |
|:----------------------|:-----------------------|:-------------------------|
| Grafana               | http://localhost:3000  | Dashboards & logs        |
| Kafka UI              | http://localhost:8080  | Kafka topic browser      |
| Prometheus (identity) | http://localhost:9091  | Identity service metrics |
| Prometheus (access)   | http://localhost:9090  | Access control metrics   |
| Loki (identity)       | http://localhost:3101  | Identity service logs    |
| Loki (access)         | http://localhost:3100  | Access control logs      |

</details>

----

<details>
<summary><strong>Project Commands</strong></summary>

| Command                                       | Makefile       | Description                                  |
|:----------------------------------------------|:---------------|:---------------------------------------------|
| `docker-compose up -d`                        | `make up`      | Start all services in detached mode.         |
| `docker-compose stop`                         | `make stop`    | Stop containers without removing them.       |
| `docker-compose down`                         | `make down`    | Stop and remove containers and networks.     |
| `docker-compose down && docker-compose up -d` | `make restart` | Restart the entire infrastructure.           |
| `docker-compose logs -f kafka`                | `make logs`    | Stream Kafka logs in real time.              |
| `docker network ls`                           | `make net`     | List networks (verify `kafka-net` exists).   |
| `docker exec -it kafka /bin/bash`             | `make exec`    | Open a shell inside the Kafka container.     |
| `docker-compose down -v`                      | `make clean`   | Remove containers along with volumes.        |
| `docker system prune -f`                      | `make prune`   | Global cleanup of unused Docker resources.   |

</details>

----
