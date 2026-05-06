# infra

This repository contains the configuration for deploying shared messaging and observability infrastructure for LIAM services. It provides Apache Kafka for inter-service communication along with Grafana and Prometheus for monitoring.

----

<details>
<summary><strong>Local Deployment</strong></summary>

Make sure Docker and Docker Compose V2 are installed on your system before proceeding.

1. Create shared Docker networks:
```sh
make network name="monitoring-network"
make network name="kafka-net"
```
2. Generate `.env` from example:
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
<summary><strong>Network</strong></summary>

The broker is configured with **Advertised Listeners**, allowing two connection modes:

**External connection *(Host Machine)***
If your service runs locally via `npm run`:
- *Broker address:* `localhost:9092`
- *Protocol:* `PLAINTEXT`

**Internal connection *(Docker Network)***
If your service runs inside its own Docker container:
- *Broker address:* `kafka:29092`
- *Protocol:* `PLAINTEXT`

**Connecting external microservices:**
Add the following to your service's `docker-compose.yml`:
```yaml
networks:
  kafka-net:
    external: true
    name: kafka-net
```

</details>

----

<details>
<summary><strong>Commands</strong></summary>

| Command                                       | Makefile                | Description                                |
|:----------------------------------------------|:------------------------|:-------------------------------------------|
| `docker compose up -d`                        | `make up`               | Start all services in detached mode.       |
| `docker compose stop`                         | `make stop`             | Stop containers without removing them.     |
| `docker compose down`                         | `make down`             | Stop and remove containers and networks.   |
| `docker compose down && docker-compose up -d` | `make restart`          | Restart the entire infrastructure.         |
| `docker compose logs -f kafka`                | `make logs`             | Stream Kafka logs in real time.            |
| `docker exec -it kafka /bin/bash`             | `make exec`             | Open a shell inside the Kafka container.   |
| `docker compose down -v`                      | `make clean`            | Remove containers along with volumes.      |
| `docker system prune -f`                      | `make prune`            | Global cleanup of unused Docker resources. |
| `docker network create --name=$(name)`        | `make network name="*"` | Create a new docker network.               |
| `docker network ls`                           | `make networks`         | List networks (verify `kafka-net` exists). |

</details>

----

<details>
<summary><strong>UI Links</strong></summary>

| Service                     | URL                   |
|:----------------------------|:----------------------|
| Grafana                     | http://localhost:3000 |
| Kafka UI                    | http://localhost:8080 |

</details>

----
