**RU** · [EN](../../audits/2026-10-29-phase2-engine-primitives-357a09d.md)

# Этап 2 (прогресс) — подтверждение путей + примитивы engine

> Статус: **в работе** (примитивы и тесты; полная приёмка матрицы отказов ещё нет)  
> Дата: 2026-10-29  
> Родитель: `357a09d`

См. EN: таблица путей, trust note, `ResolveReleaseAsset`, `JobRegistry`, `go test ./internal/update/`.

Открыто: проводка всех apply только через Resolve+Registry; полная матрица отказов; health gate для `succeeded`.
