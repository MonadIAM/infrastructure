# Участие в разработке

Language: [EN](CONTRIBUTING.md) | **RU**

Вернуться к [README](README.ru.md)

----

<details>
<summary><strong>Git Hooks</strong></summary>

*Включить Git hooks:*

```sh
chmod +x .githooks/*
git config core.hooksPath .githooks
```

#### Допустимые префиксы коммитов и веток

`feat`, `fix`, `docs`, `refactor`, `perf`, `test`, `build`, `ci`, `chore`, `revert`.

#### Формат сообщения коммита

```text
<type>(<scope>)?[!]: <subject>
```

*Поля:*

- `<type>` — один из допустимых префиксов.
- `<scope>` — необязательная область в скобках, буквы, цифры и `-`.
- `!` — необязательный маркер несовместимого изменения.
- `: ` — обязательные двоеточие и пробел.
- `<subject>` — короткое описание.

> Длина сообщения коммита ограничена 70 символами.

*Примеры коммитов:*

- `feat: add filters to user search`
- `fix(auth): handle invalid jwt token`
- `refactor(user-service)!: drop legacy ws implementation`
- `docs: update api usage section`
- `chore(ci): tweak lint step`

#### Формат имени ветки

```text
<type>/<slug>
<type>/<scope-or-id>/<slug>
```

*Поля:*

- `<type>` — один из допустимых префиксов.
- `/<scope-or-id>` — необязательная область или идентификатор, буквы, цифры и `-`.
- `/<slug>` — короткое описание, буквы, цифры, `_` и `-`.

*Примеры веток:*

- `feat/add-user-search-filters`
- `fix/schedule-update-ws-event`
- `refactor/core-user-cleanup`
- `feat/GOTEC-404/add-notifications-for-users`

> Исключения: `dev`, `main`, `master`.

</details>

----
