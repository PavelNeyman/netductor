# Web / op UI — план аудита кнопок

**Статус:** активный чеклист для прохода оператором.  
**Область:** web UI `netductor-op serve` (sidebar, Deploy, Operate, Settings, Result, Jobs).  
**Связано:** Result dock; jobs → Deploy log (вкладка Log в Result ещё в планах).  
**Версии:** часть кнопок может подтягиваться с `/v1/catalog`; запасной список — `BTN` в `internal/operator/web/app.js`.

---

## Как проверять

1. Идти **слева направо** по sidebar.
2. На каждое действие: **ожидание** ниже ↔ факт (ок / 401 / пусто / непонятно).
3. Пометка: **оставить / убрать / переименовать / перенести**.
4. Куда ушёл вывод: **Result** / **Deploy log** / **Jobs**.

**Перед Operate:** Settings → host primary + SSH-ключ → **Tunnel Start / ensure** → **Issue session** (если 401).

Формат ответа:

```text
раздел → кнопка → факт / что неясно / оставить|убрать|переименовать|перенести
```

---

## Глобально

| UI | Назначение | Ожидание |
|----|------------|----------|
| Collapse sidebar | Только иконки | Узкий rail |
| Result (header / status bar) | Показать/скрыть Result | Dock |
| Jobs | Job’ы op (deploy, offline-prep…) | Панель справа; Refresh / Running only |
| Status bar: API / secondary / paths / channel | Сводка health | Точки/текст |
| Theme / language | Тема, EN\|RU | Без полной перезагрузки |

---

## Servers

| Действие | Ожидание |
|----------|----------|
| Карточки primary/secondary (tunnel + session) | Hostname, роль, краткие метрики |
| Клик по карточке | Детали / действия |
| Back | К списку |

Без session — пусто или offline-заглушка.

---

## Sites

| Действие | Ожидание |
|----------|----------|
| Refresh | Edge devices с primary |
| + Add site | Мастер OpenWrt (Deploy → edge) |
| Карточка сайта | Топология / провал внутрь |

---

## Deploy (Installer)

### Fleet

| Кнопка | Ожидание |
|--------|----------|
| **Fleet deploy** | Job: primary (+ opt. secondary/build); лог в **Deploy log / Jobs** |

### Primary / Secondary / Build-host

| Кнопка | Ожидание |
|--------|----------|
| **Deploy …** | Долгий job, SSH с Mac; лог в Deploy log |

### OpenWrt (edge)

| Кнопка | Ожидание |
|--------|----------|
| **Prepare offline pack** | Job `edge.offline_prep`; кэш agent/token; PARTIAL → докачка |
| **Probe device** | SSH на роутер → карточка устройства |
| Шаги мастера | Modules → Dry-run → **Provision edge** |
| **Dry-run plan** | Только план |
| **Provision edge** | Provision + reboot; лог в Deploy log |
| Storage: List / Backup / Restore / Format+restore / Mute / Unmute / Offline pack on primary | Day-2 USB/extroot |

### MikroTik (site)

| Кнопка | Ожидание |
|--------|----------|
| **Save site / push** | RSC / site shell |

### NVR

| Кнопка | Ожидание |
|--------|----------|
| Add camera / Run / … | Камеры, leases (нужен session) |

### Credentials

| Кнопка | Ожидание |
|--------|----------|
| **Collect** | Сбор учёток/ключей |

---

## Operate

### Tunnel / session (всегда сверху)

| Кнопка | Зачем | Ожидание |
|--------|-------|----------|
| **Start / ensure** | Доступ к API primary | `direct` или SSH `-L` |
| **Stop** | Снять local forward | Tunnel down |
| **Issue session** | Токен → `~/.netductor/node_session` | ok + файл |
| **Load session file** | Прочитать сохранённый токен | Session подставлен |

**Важно:** VLESS может дать `direct` к API; **session** всё равно нужна (иначе 401 на `/api/*`).

### Overview

| Группа | Зачем | Result |
|--------|-------|--------|
| health / status / overview / doctor / domain / bot / metrics* | Живость API, сводка | JSON/таблицы |
| stack status / apply / rollback / watchdog | Стек на primary | apply = обновление node |
| addons / lampac | Контейнеры | Список |
| sni / presets | Reality SNI | Текущий + пресеты |
| sessions | Operator sessions | Список |
| update / latest / gh-token | Обновления | Версии |
| release local / tags / mirror / build / status / prune | Локальные релизы | Сборка/зеркало (осторожно) |

### VPN / users

| Действие | Ожидание |
|----------|----------|
| Users / refresh links | VLESS + ссылки |
| Add / enable / disable / revoke / link | CRUD |
| Policy load/save + presets | internet + services |
| Services / Policy apply | Каталог; пересборка sing-box |

### Nodes

| Действие | Ожидание |
|----------|----------|
| nodes / self / secondary | Узлы |
| ssh-hosts / clear | SSH hosts |
| mTLS certs / rotate / revoke | Серты agent plane |
| sites / push / rsc | Сайты |
| sec-cmd / sync / exit | Secondary |
| hostname / journal / restart | Диагностика |

### Edge (day-2)

| Действия | Ожидание |
|----------|----------|
| pending / devices / metrics / act / bind / cmd / guest / template / recovery / backup / export / set-site / vpn / policy | Enroll и day-2 |

### NVR / media

| Действия | Ожидание |
|----------|----------|
| cams, leases, live, PTZ, rec, clip, wifi, wizard | Если NVR включён |

### Git / registry / build

| Действия | Ожидание |
|----------|----------|
| projects, ship, migrate, mirror, vps-queue, build-host, artifacts | CI primary/build-host |

### Backup / DNS / probes / incidents

| Действия | Ожидание |
|----------|----------|
| backup schedule, DNS, cleanup, probes, audit, devices, canary, smoke, incident | Расписания и инциденты |

### SSH allowlist / reboot

| Действие | Ожидание |
|----------|----------|
| Inventory / probe / allowlist get·save·merge·enforce | Ключи SSH |
| Reboot challenge → confirm | Двухшаговый reboot |

### Rooms

| Действие | Ожидание |
|----------|----------|
| List / upsert / delete / photo | Site rooms |

### Advanced

| Действие | Ожидание |
|----------|----------|
| Method + path + body **Send** | Сырой `/api/...` |

---

## Settings

| Элемент | Зачем |
|---------|-------|
| Primary / VPN host / prefer VPN | SSH и prefer_vpn |
| Key, user, API port/base | Tunnel + session |
| Secondary host | Prefill форм |
| **Save** | `operator.conf` + localStorage |
| GitHub token Save / Clear / Refresh | Токен на primary |

---

## Result dock

| UI | Ожидание |
|----|----------|
| Output | Таблицы/summary последнего API |
| History | Прошлые вызовы |
| Position bottom / right / top | Справа только **⋮** |
| Mode panel / card / both | Dock и/или inline |
| JSON / Clear / Hide | Сырой JSON; очистка; скрыть |

**Пока нет:** вкладка **Log** для stdout job’ов (offline-prep, provision) — сейчас **Deploy log** + **Jobs**.

---

## Минимальный порядок прохода

1. Settings + Tunnel + Session  
2. Servers / Sites (вид)  
3. Operate → Overview: health, doctor, status, metrics  
4. VPN: users + policy load  
5. Nodes: secondary status  
6. Одна длинная задача: offline-prep → Jobs + Deploy log  
7. Остальное — только то, чем пользуетесь; остальное → «убрать / в Advanced»

---

## Якоря в коде

| Что | Где |
|-----|-----|
| Статические кнопки Operate | `BTN` в `internal/operator/web/app.js` |
| Особые формы | `special[...]` |
| Каталог с primary | `GET /v1/catalog` |
| Длинные job’ы | `streamPost` → `/v1/jobs` |
| Deploy log | `#log`, `#deployLogSec` |
| Result | `#resultDock`, `showControl` |

---

## После аудита зафиксировать

- Кнопки **убрать** или спрятать в Advanced  
- Переименования (RU/EN)  
- Склейка групп (например release/* только под Updates)  
- Поломки вёрстки (скрин + раздел)  
- Подтверждение дизайна Log в Result (когда будем внедрять)
