# GitOps и обновления

Language: [EN](GITOPS.md) | **RU**

Вернуться к [README](../README.ru.md)

## Где живут образы

Zot развёрнут как `deployment/monadiam-registry` в `monadiam-data`.

На узле он доступен по `http://127.0.0.1:5000/v2/`, внутри кластера — через `monadiam-registry.monadiam-data.svc:5000`.
Данные находятся в `/var/lib/monadiam/zot/data`, правила зеркалирования и retention — в [config.json](../clusters/dev/infrastructure/zot/base/config.json).

Прикладные образы должны существовать по digest из `clusters/dev/applications/*/kustomization.yaml`.
Для пустого Zot сначала запустите MicroK8s и registry, затем опубликуйте образы.

Пример для уже собранного образа:
```sh
just k8::start
just zot::apply
just zot::push monadiam/identity-service:dev monadiam/identity-service:dev
```

После публикации приведите tag/digest overlay к фактическому образу.
Универсальной команды загрузки всех прикладных образов на пустой Zot в этом
репозитории нет.

## Синхронизация приложений

Argo CD читает `clusters/dev/argocd` из удалённой ветки `dev`.
У Applications включён `selfHeal: true`, автоматический prune отключён.
Ручное изменение управляемого поля будет возвращено к Git-конфигурации.
Удалённый из Git manifest нужно отдельно удалить из кластера.

Image Updater проверяет semver tags под `linux/amd64` и создаёт PR
с изменением Kustomize в ветку `dev`. Обновления включаются label
`monadiam.io/image-updates: enabled` на Application.

Для GitHub App задайте в `.env.dev` ID приложения, installation ID и путь к private key:

```sh
just gitops::credentials
```

Эта команда создаёт Secret Image Updater. Доступ Argo CD к закрытому Git-репозиторию
настраивается отдельно.

| Изменение                            | Применение                                        |
| :----------------------------------- | :------------------------------------------------ |
| Terraform resource                   | Plan нужного root, затем apply                    |
| Helm values / платформенный manifest | Reconcile соответствующей фазы                    |
| Образ или конфигурация приложения    | Коммит в `dev`, Argo CD sync, проверка rollout    |
| Host config / systemd unit           | Установка обновлённого файла и перезапуск сервиса |
| Vault policies, roles, keys          | `just vault::plan`, затем `just vault::apply`     |

Для отката приложения верните предыдущие image/config значения в Git.
Старый digest должен оставаться доступным в registry. Миграции БД, события
и ключи Vault не откатываются вместе с образом.

Registry использует on-demand mirrors, GC и retention. Теги `keep-dev-*`
и `keep-prod-*` исключены из обычного удаления старых тегов.
