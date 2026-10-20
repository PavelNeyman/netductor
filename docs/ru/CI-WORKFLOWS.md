# CI workflows (netductor VPS)

## Resolve path (зафиксировано)

| Правило | |
|--|--|
| Каталог | **только** `.github/workflows/` |
| Файл + `name:` | **= имя проекта = имя GitHub repo** |
| Resolve | точный `.github/workflows/<Name>.yml` или `.yaml` |
| Папка `ci/` | **удалена** — не резолвится |
| `git project add` | Name = repo name; workflow path автоматически |
| host | `vps` (default) на primary; `mac` → очередь + TG/CLI |
| Windows | `GOOS=windows GOARCH=amd64` в linux container (Go); без MSI/WiX на VPS |

```bash
netductor git project build niimbot-ios v1.2.3
netductor git project queue list
netductor git project queue done <id>
```

```bash
netductor-op project build niimbot-ios --ref v1.2.3
```

---

# Netductor CI (подмножество GHA)

**Runner:** primary VPS, Docker/Podman (`internal/ci` + `internal/gha`). Это не полный GitHub Actions.

## Выбор workflow

1. Явный `workflow` у project
2. Иначе первый `.github/workflows/*.{yml,yaml}` (сортировка)
3. Иначе shell pipeline (`--pipeline …`)
4. Иначе ошибка

Рекомендация: `.github/workflows/ci.yml`.

## YAML

- `jobs.*.steps[].run`, `container`, `env`, `working-directory`
- `uses: actions/*` — skip (checkout уже у netductor)
- `on:` / `runs-on` — игнорируются
- Jobs в алфавитном порядке имён

```yaml
name: ci
on: [push]
jobs:
  test:
    runs-on: ubuntu-latest
    container: golang:1.22-bookworm
    steps:
      - name: test
        run: |
          set -euo pipefail
          go test ./...
```

```bash
netductor git project add myapp org/myapp
netductor git project sync myapp
netductor git project build myapp
```

## Изоляция

Сборка в контейнере; `NETDUCTOR_CI_HOST=1` — WARN в doctor. PAT не в yaml репозитория.

## Mac / Windows

| Цель | Где |
|--|--|
| Linux | primary container |
| Windows exe | cross-compile на VPS |
| Xcode/ipa | host=mac + queue |

## Troubleshooting

Нет workflow — путь и имя project. Mac queue — `queue list` / `done` после локальной сборки.
