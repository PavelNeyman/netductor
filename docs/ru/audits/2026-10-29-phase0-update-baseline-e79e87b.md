**RU** · [EN](../../audits/2026-10-29-phase0-update-baseline-e79e87b.md)

# Этап 0 — Базовый аудит системы обновлений

> Статус: **итоговый** (только разбор исходников; apply на production в этом аудите не выполнялся)  
> Дата: 2026-10-29  
> Репозиторий: `PavelNeyman/netductor`  
> Базовый commit: `e79e87b07999f9e2de3d129de4489bbbe531e2ff`  
> Релиз/тег: **v0.9.596** (`VERSION` = `0.9.596`)  
> Область: карта точек входа обновлений, матрица артефактов, доверие/откат, соглашения TG/Web, открытые вопросы  
> Пара: [EN](../../audits/2026-10-29-phase0-update-baseline-e79e87b.md)

## 1. Краткий вывод

- **Оценка:** уже есть **несколько независимых** путей обновления (node через GitHub releases, operator self-update, Edge `agent_update`, ship/join secondary, addons, local build). **Единого каталога компонентов и общей state-machine job** нет.
- **Риск:** доверие в основном к **SHA-256 из того же GitHub Release**, что и бинарник (нет отдельной подписи происхождения).
- **Подтверждено:** карта §5; матрица assets v0.9.596; agent_update с confirm=yes, ELF, `.prev` rollback; скрипты release/verify/check-versions.
- **Не проверялось здесь:** повторный live `netductor update apply` на prod; ранее в ops отмечался PATH `netductor-agent` 0.9.151 при node 0.9.596 на secondary.

## 2. Метод

- Исходники: `cli_misc.go` (`runUpdate`), `api_update.go`, `internal/update/*`, `agent_ops.go`, `cli_edge.go`, `netductor-op/cli_update.go`, TG handlers versions/updates, `scripts/release.sh`, `verify-release-binaries.sh`, `check-versions.sh`, GHA release workflow, план UPDATE-SYSTEM.
- Команды: `git rev-parse HEAD`, `cat VERSION`, поиск по дереву. **`go test` / live apply — не запускались.**

## 3. Статус проверок

| Проверка | Команда | Результат |
|---|---|---|
| HEAD | `git rev-parse HEAD` | `e79e87b07999f9e2de3d129de4489bbbe531e2ff` |
| VERSION | `cat VERSION` | `0.9.596` |
| Тег | `git tag -l` | `v0.9.596` |
| go test / live apply | — | **не выполнялись** |

## 4. Базовая идентичность

main HEAD = `e79e87b…` (merge PR #1). Релиз: [v0.9.596](https://github.com/PavelNeyman/netductor/releases/tag/v0.9.596). Operator только **darwin**.

## 5. Карта точек входа

| ID | Инициатор | Цель | Выбор asset | Целостность | Атомарность | Рестарт/health | Откат |
|---|---|---|---|---|---|---|---|
| U-CLI-NODE | `netductor update [apply]` | node/tg/agent | tag + component + arch | SHA256SUMS, опция `--skip-verify` | internal/update | опциональный restart | backup, если не `--no-backup` |
| U-CLI-CHECK | `update check\|list` | только чтение | список releases | GitHub API | — | — | — |
| U-API | `/api/update/*` | тот же движок | как CLI | session | как CLI | как CLI | как CLI |
| U-TG | меню Updates | оператор → backend | compare latest | token status | не второй движок | UI | ограничен |
| U-WEB | Web → `/api/update/*` | то же | то же | session | то же | UI | то же |
| U-EDGE-CMD | `edge cmd … agent_update` | agent на устройстве | arch + SHA256SUMS | confirm=yes, SHA, ELF | stage + `.prev` + rename | процесс agent | rollback `.prev` |
| U-OP | netductor-op update | Mac op | darwin assets / brew | checksums / brew | self-replace | restart op | brew special case |
| U-SEC-DEPLOY | deploy/join secondary | node на S | ship | SSH + artifact | install path | units | snapshot/re-deploy |
| U-ADDONS | addons update | lampac/registry/git | своё | разное | разное | разное | разное |
| U-CI-REL | release.sh / GHA | публикация | multi-arch | SHA256SUMS, tag==HEAD, post SHA | publish | — | republish |
| U-LOCAL | local releases / build host | зеркало | local SUMS | file SHA | import | — | prune |

## 6. Матрица артефактов v0.9.596

| Логический ID | Шаблон asset | Роль |
|---|---|---|
| `node` | `netductor-linux-{amd64,arm64}` | Primary / Secondary node |
| `tg` | `netductor-tg-linux-{amd64,arm64}` | Telegram bot |
| `agent` | `netductor-agent-linux-{amd64,arm64,arm,mipsle,riscv64}` | Edge / agent |
| `op` | `netductor-op-darwin-{arm64,amd64}` | Mac operator |
| manifest | `SHA256SUMS` | все |

**Пробел:** нет стабильного API «component + instance + job» для всех целей.

## 7. Доверие

SHA256SUMS рядом с assets; `--skip-verify`; Edge confirm=yes + ELF; release.sh: clean tree, tag=HEAD, проверка SHA после upload. Подписанного манифеста нет.

## 8. Telegram / Web

- TG: `m:versions`, `m:updates*`, кнопки, back в `m:menu`; токен GitHub; не плодить вторые шаблоны карточек (этап 5).
- Web: session + `/api/update/*`; i18n/check-web-ui; этап 6 — только design-system.

## 9. Находки

| ID | Суть | Severity | Статус |
|---|---|---|---|
| P0-01 | Несколько движков без общей job-модели | high | confirmed |
| P0-02 | Provenance = SHA того же канала | medium | confirmed |
| P0-03 | `--skip-verify` | medium | confirmed |
| P0-04 | Неоднозначность agent vs node version на secondary | medium | hypothesis / ops |
| P0-05 | Addons вне ядра unified UI v1 | low | confirmed |

## 10. Открытые вопросы

1. Day-2 update Secondary node — только локально на S или job с Primary?  
2. `netductor-agent` на secondary — обязательный unit или наследие?  
3. Полный перечень Web-страниц, бьющих в `/api/update/apply`.  
4. Минимальная from-version для Edge agent_update.  
5. Может ли TG apply без двойного подтверждения на Primary?

## 11. Приёмка этапа 0

Карта потоков, матрица артефактов, соглашения UI, список вопросов — **есть**. Изменения кода для этапа 0 **не требуются**.

## 12. Дальше

**Этап 1** — identity и контракт обновления; не отмечать выполненным без схемы/тестов и пары EN/RU в том же изменении.
