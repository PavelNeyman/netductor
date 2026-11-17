# План: устойчивость флота и эволюция к mesh

**Статус:** принятое направление 2026-11-17  
**Связано:** [ADR-ROLE-REGION.md](../ADR-ROLE-REGION.md), [ARCHITECTURE-FREEZE.md](../ARCHITECTURE-FREEZE.md), CF-14

## Цель

Эволюция от схемы «один control + один egress» к **устойчивому флоту**:

- **роль** ноды — функциональная, не географическая;
- на ноде может быть **разная нагрузка**, не только VPN;
- отказ одной машины не обязан валить data-plane (и позже — управление);
- путь к более сильному mesh **не закрыт**, но старт **не** с multi-master SoT.

## Явные non-goals

| Пункт | Зачем |
|-------|--------|
| Multi-master control (пишут все) в фазах 0–3 | Split-brain, конфликты policy/CA |
| Разовый rename всех Go-пакетов | Совместимость; хватает alias |
| Build на каждой ноде | Build = capability |
| Multi-master как обязательный финал | Часто достаточно active/standby + multi-egress |

## Модель

### Три оси

| Ось | Смысл | Примеры |
|-----|--------|---------|
| **Control-ness** | Кто держит SoT кластера | `control`, `member` |
| **Capabilities** | Что можно запускать | `vpn_ingress`, `vpn_egress`, `dns`, `media`, `build`, `nvr` |
| **Region** | Ярлык размещения | `ru-msk`, `fi-hel`, `us-nyc` |

**Роль (совместимость CF-14):** `control`≡`primary`, `egress`≡`secondary`, плюс `build`, `edge`.

### География

- **Control ≠ «зарубежье».** Control — плоскость управления (API, policy, users, orchestration). Может быть в любой зоне, где приемлемы секреты и доступность API.
- **Egress ≠ «только РФ».** Точка пользовательского VPN / путь трафика; регионов может быть много.
- Шаблон «control в EU + egress в RU» — **рекомендация**, не определение ролей.

### Region

```text
region ::= <iso3166-1-alpha2> "-" <city-slug>
```

Примеры: `ru-msk`, `fi-hel`, `nl-ams`. Код страны — ISO alpha-2. Несколько нод в одном городе — один `region`, разные peer id.

### Hostname

```text
nd-<role>-<region>    # nd-control-fi-hel, nd-egress-ru-msk
```

### State

Кластерные документы со временем: `revision`, `updated_at`, `writer_id` — уже при одном writer, чтобы HA не ломал формат.

## Фазы

### Фаза 0 — Документы

- [x] Этот план + уточнение ADR.
- [x] FREEZE/ARCHITECTURE: control не определяется страной.
- [x] Глоссарий UI/docs.

### Фаза 1 — Имена, region, capabilities

Region на peer/node, capabilities (хотя бы флаги), подписи в sub, canonical в UI, подсказки hostname в мастерах. Пустой region = как сейчас.

### Фаза 2 — Data-plane (multi-egress)

Несколько egress, health, sub без мёртвых peers, runbook добавления/вывода.  
**Это первый mesh, который видит пользователь VPN.**

### Фаза 3 — Control HA (active/standby)

Реплика SoT, promote, список control endpoints у agent, fencing.

### Фаза 4 — Node API везде, запись только у leader

Локальные read/day-2; мутации SoT — на держателе lease.

### Фаза 5 — Multi-master (опционально)

Только отдельное решение после 2–4. По умолчанию **не** планировать.

## Порядок

```text
0 → 1 → 2 → 3 → 4 → (5?)
```

Фаза 2 не ждёт 3. Фаза 5 не обязательна.

## Прогресс

| Фаза | Статус |
|------|--------|
| 0 Документы | в работе |
| 1 Schema | не начата |
| 2 Multi-egress | частично (peers/slots; довести health→sub) |
| 3 Control HA | не начата |
| 4 Node API | не начата |
| 5 Multi-master | отложена |


## Обновления и сведение конфигурации (обязательно)

См. [POLICY-UPDATE-CONVERGENCE.md](POLICY-UPDATE-CONVERGENCE.md).

- Оператор **не** ставит каждый релиз: миграции и floors обязаны переживать skip-version.
- Смена стандартов (hostname `nd-<role>-<region>`, region, capabilities, baseline) → **миграция / host-baseline**, не только доки; существующие ноды и роутеры сходятся после update.
- Работа по этому плану: коммиты в `main` **без** обязательного GitHub Release на каждый шаг; релиз — пакетом или когда без нового бинаря нельзя безопасно обновиться.

## Sub / VLESS: region в имени профиля (зафиксировано 2026-11-17)

- В fragment URI — **region + флаг** (`#🇷🇺-ru-msk`), **роль не указывать**.
- Источник: поле `region` на peer (оператор: деплой / `secondary region`); пусто → legacy IP-label.
- Hostname `nd-*-ru-msk` может **подсказать** region для label (и migrate backfill), IP-geo только suggest в UI.
- Обновление без интерактива: не спрашивать в stack apply; doctor / day-2 set.
- Смена только fragment: sub refresh подхватывает имя; UUID/endpoint не трогаем.
