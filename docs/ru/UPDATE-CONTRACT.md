**RU** · [EN](../UPDATE-CONTRACT.md)

# Единый контракт обновлений (этап 1)

> Статус: **принят для этапа 1** (типы и тесты переходов в дереве; полный engine — этап 2)  
> Дата: 2026-10-29  
> Код: `internal/update/contract.go`, `contract_test.go`  
> Этап 0: [audits/2026-10-29-phase0-update-baseline-e79e87b.md](audits/2026-10-29-phase0-update-baseline-e79e87b.md)

## 1. Цель

Единая **логическая** модель «что обновляется где» для CLI, API, Telegram и Web.  
Клиенты **не** придумывают имена release-asset и **не** пишут второй движок download/verify. Они передают **component + instance + target version** (или latest compatible); asset выбирает backend через `AssetName` и метаданные релиза.

Текущие реализации (`netductor update`, Edge `agent_update`, op self-update, deploy ship) остаются до этапа 2.

## 2. ID компонентов

| ID | Семейство бинарников | Типичная роль |
|----|----------------------|---------------|
| `node` | `netductor-linux-*` | Primary или Secondary **хост** |
| `tg` | `netductor-tg-linux-*` | Telegram-бот |
| `agent` | `netductor-agent-linux-*` | Edge; опционально agent на Secondary |
| `op` | `netductor-op-darwin-*` | Только Mac operator |

**Вне каталога v1:** addons — отдельные API.  
**Экземпляр:** `instance_id` + `role` (`primary`\|`secondary`\|`edge`\|`operator`) + `platform`.

Job всегда на пару **(component, instance_id)**.

## 3. Поля `ComponentInstance`

component, instance_id, role, platform, installed_version, running_version, build_commit, desired_version, available_version, state, health_ok, last_error, updated_at_unix.

UI не показывает desired как «running» до `state=succeeded` после health check.

## 4. Состояния `JobState`

См. EN и `CanTransition` в коде: idle→checking→available→queued→downloading→verifying→installing→restarting→health_checking→succeeded; ветки failed / rollback_* / cancelled.

## 5. Операции backend

catalog, detail, check, compatible, start, job, history, rollback — как в EN.  
Идемпотентность по `idempotency_key`; один активный job на (component, instance); timeout 900s; stale UI 300s.

## 6. Авторизация

CLI — локальный оператор; HTTP — session; TG — allowlist + server-side re-check; секреты не в callback.

## 7. Соответствие текущему коду

Как в EN §7 (Phase 0 map).

## 8. Тесты

`go test ./internal/update/ -count=1`

## 9. Вне этапа 1

HTTP catalog handlers, подпись манифеста, rename edge binary, bulk fleet update.

## Бесшовность (ops)

- **CLI flags:** могут меняться, если этого требует engine; updates must still apply cleanly on hosts that last ran an older CLI (document breaking flags in CHANGELOG).
- **Config:** migrate when development changes on-disk formats; do not assume "no migration" forever.
- **UI:** do not break existing TG/Web surfaces; **add** UI when new update features ship (catalog/job status), using existing templates and i18n.
- **Engine under the hood:** tg/agent host apply uses Resolve + JobRegistry + atomic install + health; node remains `stack.Apply` until Phase 2 wiring completes.

