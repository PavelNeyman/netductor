# Политика: обновления, миграции и сведение к desired state

**Статус:** обязательно для агентов и контрибьюторов · **2026-11-17**  
**Связано:** [UPDATE-CONTRACT.md](../UPDATE-CONTRACT.md), `internal/migrate`, CF-15 floors, [PLAN-FLEET-MESH-EVOLUTION.md](PLAN-FLEET-MESH-EVOLUTION.md)

## 1. Реальность оператора

- Не ставится каждый промежуточный тег.
- Возможен прыжок `0.9.X → 0.9.X+N`.
- Каждый merge в `main` обязан сохранять skip-version upgrade **или** явный **upgrade floor**.

## 2. Бинарники (CF-15)

Ломающее изменение формата/порядка apply → в том же коммите `DeclareBinaryFloor`.  
Идемпотентные миграции → без floor.  
`upgrade.json` / preflight блокируют local `< min_from`.

## 3. Миграции (`internal/migrate`)

`Required` → `Apply` → `Verify`; журнал; идемпотентность; **учёт drift** (ручные правки, старые ноды); role-aware no-op.

## 4. Desired state (подход IaC) — обязательно

После обновления нода/роутер **сходятся** к стандарту из кода и доков, а не только запускают новый бинарь.

Примеры: hostname по конвенции, role file, firewall/SSH baseline, LE hooks, schema peer/region, edge templates.

Детект отклонения → безопасный автофикс → Verify; иначе явный doctor/migrate fail.

VPS и OpenWrt в scope managed path.

Опасные переименования (DNS/LE) — отдельный migration id и precheck.

## 5. Бесшовность

Старый CLI/host → чистый apply. Смена форматов → migrate. UI не ломать, новые фичи — добавить. Один движок download/verify.

## 6. Релизы vs main

Обычная работа по плану: **коммиты без релизов**.  
Релиз — пакетом в конце фазы / по запросу / когда без нового бинаря нельзя продолжать (floor, broken migrate path).

Агенты не считают, что прод уже на tip `main`.

## 7. Чеклист перед merge

Миграция или baseline · Required на legacy · floor при необходимости · doctor · EN/RU · без «обязательно каждый тег» · без лишнего release.
