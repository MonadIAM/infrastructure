# GitOps and updates

Language: **EN** | [RU](GITOPS.ru.md)

Back to [README](../README.md)

## Image registry

Zot runs as `deployment/monadiam-registry` in `monadiam-data`. The host reaches it at `http://127.0.0.1:5000/v2/`. Inside the cluster, use `monadiam-registry.monadiam-data.svc:5000`. Data lives in `/var/lib/monadiam/zot/data`. Mirror and retention rules are in [config.json](../clusters/dev/infrastructure/zot/base/config.json).

Application images must exist at the digests in `clusters/dev/applications/*/kustomization.yaml`. If Zot is empty, start MicroK8s and the registry before publishing images.

For an image that has already been built:

```sh
just k8::start
just zot::apply
just zot::push monadiam/identity-service:dev monadiam/identity-service:dev
```

After publishing, align the overlay tag and digest with the actual image. This repository has no single command to load every application image into an empty Zot.

## Application sync

Argo CD reads `clusters/dev/argocd` from the remote `dev` branch. Applications use `selfHeal: true`, while automatic prune is disabled. Argo CD restores manually changed managed fields to their Git values. Manifests removed from Git must be deleted from the cluster separately.

Image Updater checks semver tags for `linux/amd64` and opens a PR with Kustomize changes against `dev`. The `monadiam.io/image-updates: enabled` Application label enables updates.

Set the GitHub App ID, installation ID, and private key path in `.env.dev`, then run:

```sh
just gitops::credentials
```

This creates the Image Updater Secret. Argo CD access to the private Git repository is configured separately.

| Change                             | Apply it with                                    |
| :--------------------------------- | :----------------------------------------------- |
| Terraform resource                 | Plan and apply the relevant root                 |
| Helm values / platform manifest    | Reconcile the relevant phase                     |
| Application image or configuration | Commit to `dev`, sync Argo CD, and check rollout |
| Host config / systemd unit         | Install the updated file and restart the service |
| Vault policies, roles, keys        | `just vault::plan`, then `just vault::apply`     |

To roll back an application, restore its previous image and configuration in Git. The old digest must still be available in the registry. Database migrations, events, and Vault keys do not roll back with an image.

The registry uses on-demand mirrors, GC, and retention. Tags matching `keep-dev-*` and `keep-prod-*` are excluded from normal old-tag deletion.
