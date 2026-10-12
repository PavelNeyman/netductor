# Быстрый профиль тестов (~2 GiB)

**Статус:** актуально (2026-10-12).  
Для хостов ~2 GiB без swap (полный `go test ./internal/...` убивает среду).

## Рекомендуемое окружение

```bash
export GOTOOLCHAIN=local
export GOPROXY=https://proxy.golang.org,direct
export GOMAXPROCS=1
export NETDUCTOR_DISABLE_VPS_WORKER=1
```

## Что гонять

Пакет за пакетом (или маленькими батчами):

```bash
go test ./internal/notify -count=1 -timeout 60s
go test ./internal/incident -count=1 -timeout 60s
go test ./internal/session -count=1 -timeout 60s
go test ./internal/audit -count=1 -timeout 60s
go test ./internal/fleet -count=1 -timeout 60s
# ... остальные лёгкие пакеты
```

## Пропускать / следить

- `./internal/incident` collect-тесты могут timeout — не блокер.
- `./internal/install` может OOM на слабых хостах.
- Полный suite одним процессом на ~2 GiB не поддерживается.

## Заметки

- Фикс бесконечного цикла worker уже в main (`NETDUCTOR_DISABLE_VPS_WORKER=1`).
- После зелёного батча + doctor (включая проверку VERSION vs tag) обычно достаточно.
- На CI или больших хостах нормальный `go test ./...` остаётся предпочтительным.
