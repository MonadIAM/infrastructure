# @monadiam/infra

Local MonadIAM infrastructure for `k3d`/`k3s`.

----

<details>
<summary><strong>Local Deployment</strong></summary>

1. Install required tools:

```sh
docker
k3d
kubectl
helm
openssl
jq
```

2. Generate local environment file:

```sh
make env
```

3. Start the cluster and reconcile the platform stack:

```sh
make start
```

4. Check current state:

```sh
make status
make vault-status
```

</details>

----

<details>
<summary><strong>Project Commands</strong></summary>

| Makefile            | Description                                      |
|:--------------------|:-------------------------------------------------|
| `make env`          | Generate `.env` from `.env.example`.             |
| `make start`        | Create or start the local k3d cluster and stack. |
| `make resume`       | Reconcile stack against an existing cluster.     |
| `make stop`         | Stop the local k3d cluster.                      |
| `make restart`      | Restart the local cluster and reconcile stack.   |
| `make status`       | Show cluster, workloads, and platform status.    |
| `make vault-status` | Show Vault initialization and seal status.       |

</details>

----
