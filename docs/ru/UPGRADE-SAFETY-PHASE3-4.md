# Безопасное обновление — Phase 3/4 и релиз v0.9.756

## Что на серверах

Опубликованный релиз GitHub: **v0.9.756**.  
Коммиты Phase 3/4 на `main` могут быть **впереди** тега — до следующего релиза на VPS только поведение **0.9.756**.

## Обновление до текущего *опубликованного* релиза

Уже 0.9.756 — ничего делать не нужно (кроме желаемого secondary upgrade).

Если старше:

```bash
netductor stack apply v0.9.756
netductor secondary cmd <id> upgrade:v0.9.756
```

Mac: `brew upgrade netductor-op` / `netductor-op update`.

## Phase 3/4 до включения

Без файла `control_role`, без `SERVERS=`, без timer — поведение как у одного control. Железо для merge кода не требуется.

Включать standby/cutover — только когда будете тестировать HA, по runbook.

## Связанное

- [RUNBOOK-CLUSTER-STATUS.md](RUNBOOK-CLUSTER-STATUS.md)
- [RUNBOOK-CONTROL-CUTOVER.md](RUNBOOK-CONTROL-CUTOVER.md)
