# План: действительно тонкий UI (jobs на бэкенде)

**Статус:** Phase 0–1 в v0.9.408 · **2026-10-17** · Полная EN-версия: [PLAN-THIN-UI-JOBS.md](../PLAN-THIN-UI-JOBS.md)

## Цель

Долгие операции живут в **бэкенде**. UI (Web, TUI, Telegram, CLI) только:

1. стартует работу → `job_id`
2. опрашивает статус / лог / результат
3. может отменить
4. после refresh / перезапуска UI **подключается к тому же** `job_id`

## Сейчас

| Где | Проблема |
|--|--|
| **netductor-op** (Mac) | Edge probe/provision, Fleet, deploy VPS — внутри HTTP/TUI-запроса; refresh убивает прогресс |
| **Primary** | Очереди агентов (edge/secondary/build) уже async; часть bulk (stack/git) ещё sync в хендлере |
| **TG** | Часто enqueue (ок); длинные callback без job_id — риск |

## Два store

- **OperatorJobStore** — LAN SSH (factory OpenWrt, первый deploy с Mac)
- **PrimaryJobStore** — всё, что делает VPS / очереди агентов

Один контракт: `POST/GET/cancel /v1/jobs` (op) и `/api/jobs` (primary).


## Модель процесса op (TUI / Web / CLI)

**Решение:** TUI **не** выносим в отдельный бинарь. Jobs держит **`netductor-op serve`**; TUI/Web/CLI — клиенты `127.0.0.1:7373`.

| Процесс | Роль |
|--|--|
| `netductor-op serve` | `OperatorJobStore`, SSH, probe, provision |
| `netductor-op tui` | только API jobs (без своего SSH на долгих путях) |
| Браузер / CLI | тот же serve |

Закрытие TUI или refresh Web **не** отменяет job. Опциональный launchd — позже. Primary jobs — на VPS.

## Фазы

0. `internal/jobs` + API на op и primary  
1. **edge.probe / provision / offline-prep / fleet** на op + reattach в Web  
2. Primary: stack, git build, backup…  
3. Чеклист **всего** функционала UI (не только edge)  
4. UX: список jobs, TG edit, cancel + context в SSH  

## Критерий успеха

Refresh на probe **не** ставит USB-пакеты второй раз; ответ старта job &lt; 500 ms; все медленные действия — job; day-2 edge по-прежнему через `EnqueueCmd`.
