# Бэклог правок кода после деплоя Secondary B (2026-11)

**Статус:** открыт  
**EN:** [PLAN-CODE-FIXES-AFTER-B.md](../PLAN-CODE-FIXES-AFTER-B.md) — полный текст; этот файл держит паритет по пунктам.

Контекст: B `195.140.147.20` (peer-195-140-147-20, slot 1) задеплоен и **promoted** 2026-11-06; A в `ready` на окно наблюдения.

## Приоритеты (кратко)

| P | ID | Тема |
|---|-----|------|
| P0 | CF-1 | TCP probe: не ok через локальный `ss` |
| P0 | CF-2 | Watchdog: не рестартить sing-box только из-за SP/PS |
| P1 | CF-3 | Slot-aware agent/uplink VIP |
| P1 | CF-4 | Prepare vs host-baseline UFW :22 |
| P1 | CF-5 | Зачистка hoster firewall на first deploy |
| P2 | CF-6 | Harden re-deploy |
| P2 | CF-7 | Integrity drift после ручной подмены бинаря |
| P2 | CF-8 | Тестовый долг |
| P1 | CF-9 | pullAndApply: ошибка restart sing-box |
| P1 | CF-10 | ApplyConfig: ошибки restart/bump |
| P2 | CF-11 | bumpSecondaryConfigVer silent no-op |
| P1 | **CF-12** | **APT Guard policy под Debian 13 (Trixie)** |
| P1 | **CF-13** | **Разбор и починка `stack apply` / нормального update path** |
| P2 | **CF-14** | **Имена: role control/egress + region (гео в sub)** |
| P1 | **CF-15** | **Upgrade floors: `min_from` из кода + CI (ступеньки только при break)** |
| P1 | **CF-16** | **PeerRemove: UFW 8789 comment `netductor-api-peer`** |

## CF-12 — policy под Trixie

На primary 2026-11-06 после baseline: `aptguard` FAIL policy_v=1, 8 findings — `POLICY_MISMATCH` / `MISSING_REQUIRED` при живых `trixie` / `trixie-security` / updates. Это рассинхрон **default policy**, не «сломанный APT».  
**Сделать:** default/migration policy для Trixie; required `debian-main`/`debian-security` узнают актуальные suite; doctor без FAIL на stock mirrors.  
**Не:** `aptguard apply` только чтобы убрать FAIL.

## CF-13 — stack apply

Primary обновляли **ручной подменой бинаря**, потому что `stack apply` не проходил.  
**Сделать:** аудит причин отказа, fail-closed стадии, transaction/recovery, VERSION+integrity после успеха, путь обновления agent на B/A.  
**Приёмка:** `stack apply <tag>` на здоровом primary с release assets либо успех, либо ясная стадия + recovery; без рутины scp бинарей.

Полные acceptance и fix-direction — в EN-документе.


## CF-15 — upgrade floors (ступеньки только при break) — **в работе 0.9.692**

Registry + preflight gate; автоцепочка ещё нет. EN § CF-15.


**Проблема:** при прыжке через много тегов иногда нужен промежуточный бинарь (сломан on-disk format, опасный порядок apply, peer/slot material без recover). Ручная галочка `min_from` в релизе **будет забыта**.

**Решение:**
1. Floor объявляется **в коде** в том же коммите (`DeclareBinaryFloor` / аналог) — только для реальных break.
2. CI **сканирует** floors и **генерирует** `upgrade.json` / metadata релиза; рассинхрон → fail.
3. Update: сначала **отказ**, если local < min_from; позже — автоцепочка download/apply + migrate между шагами.

Идемпотентные миграции и обычные фичи **пола не ставят** → один прыжок + startup migrate (skip-version).

Фазы A (gate) / B (path) / C (chain) и acceptance — EN: [PLAN-CODE-FIXES-AFTER-B.md](../PLAN-CODE-FIXES-AFTER-B.md) § CF-15.

## Skip-version (кратко)

Миграции идемпотентны; policy/defaults догоняют без промежуточных тегов; integrity не спамит VERSION после multi-hop. Ступеньки бинарей — только CF-15.

## Диагностика B

См. EN: `scripts/diag-secondary-b.sh`.

## CF-14 — control / egress + region

Решение: не in/out. Роль (`control` / `egress` / build / edge) + **region** для пользователя в **sub** (eu/ru…). Aliases primary/secondary долго. Подробности — EN plan.

## CF-16 — PeerRemove 8789

Создание: `netductor-api-peer`. Удаление раньше ждало `nd-svc-sp` → refuse. Исправлено в 0.9.693 (оба комментария). После обновления primary: `netductor svc-paths peer remove <id>`.
