# Сводка для ревью — единая система обновлений (2026-10-29+)

**Живой документ.** Обновлять при каждом коммите фазы, чтобы следующий ревьюер имел одну точку входа.

## 1. Цель

Один безопасный и наблюдаемый путь обновления компонентов Netductor (node / tg / agent / op) через CLI (далее API/Web/TG), с ясной идентичностью, целостностью, health и recovery — без ломки живых Primary/Secondary на **v0.9.596** в середине миграции.

## 2. Цепочка коммитов

См. EN-версию §2 (`docs/reviews/2026-10-29-WORK-SUMMARY.md`). Актуальный tip после Phase 3 — в `git log`.

```bash
git log --oneline e79e87b^..HEAD
go test ./internal/update/ -count=1
bash scripts/check-versions.sh
bash scripts/check-update-asset-matrix.sh
```

## 3. Статус фаз

| Фаза | Статус | Доказательства |
|------|--------|----------------|
| 0 Аудит | **готово** | `docs/ru/audits/2026-10-29-phase0-…` |
| 1 Контракт | **готово** | `UPDATE-CONTRACT.md`, `contract.go` |
| 2 Engine | **готово** | ApplyHost, Resolve, JobRegistry, failures, recovery, node job |
| 3 Релиз/версии | **готово** | `VERSION-SEMANTICS.md`, `manifest.go`, CI-скрипты |
| 4+ Edge/UI/rollout | открыто | |

## 4. Бесшовность

- Флаги CLI **можно** менять; обновление хоста должно проходить.
- Конфиги **мигрируем**, если формат меняется.
- UI не ломаем; **дописываем** под новые фичи.
- Engine tg/agent «под капотом»; node — stack + registry.

## 5. Как ревьюить

1. Эта сводка + чеклист плана.
2. `go test ./internal/update/`
3. Не путать unit-тесты с cutover / R5.

## 6. Ops того же дня

- UFW на Secondary A включён через `firewall apply secondary`.
- Isolated B / cutover — **не** объявлены готовыми.
