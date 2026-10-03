# План: политики доступа к сервисам (VPN-пользователи + роутеры OpenWrt)

**Статус:** дизайн / план реализации (код ещё не полный)  
**Аудитория:** любой агент или разработчик, который делает фичу целиком  
**Связанное:** [PORTS.md](../PORTS.md), [CLIENT-PROFILES.md](../CLIENT-PROFILES.md), [EDGE-AGENT.md](../EDGE-AGENT.md), [ARCHITECTURE-FREEZE.md](../ARCHITECTURE-FREEZE.md)  
**EN:** [../PLAN-SERVICE-ACCESS-POLICY.md](../PLAN-SERVICE-ACCESS-POLICY.md)

Полное смысловое соответствие английской версии обязательно (не дословный перевод).

---

## 1. Цель

Оператор должен выдавать **доступ к внутренним сервисам по субъекту** через существующий VLESS (и будущие пути), **галочками во всех UI**, не смешивая роутеры со списком обычных VPN-пользователей.

| Тип субъекта | Примеры | Где в UI |
|--------------|---------|----------|
| **VPN-пользователь** | operator, телефон, профиль Apple TV | Пользователи → карточка → Доступ / Политика |
| **Edge-роутер** | Cudy / OpenWrt | Роутеры / Флот → карточка устройства → Политика (отдельный раздел) |

**Не** класть peer’ы роутеров (`edge-*`) в обычный список Users. **Не** оставлять навсегда только «сырой» JSON-шаблон как единственный способ настройки: шаблоны остаются бэкенд-слиянием; для оператора — каталог + галочки.

---

## 2. Не цели (v1)

- ACL на каждое устройство внутри одного общего UUID (один UUID = одна политика).
- Доверять только клиентскому split (правила Shadowrocket) как enforcement.
- Публиковать Lampac/git/registry в публичный WAN.
- Смешивать гостевой Wi‑Fi TTL grant с этим каталогом.

---

## 3. Модель

### 3.1 Каталог сервисов (глобально, на primary)

Файл в духе `/var/lib/netductor/service-catalog.json` (точный путь — по `PATHS.md` / `ndconfig`).

Структура — как в EN-документе: `version`, список `services` с `id`, `title`, `kind` (`egress` | `internal`), `endpoints`, опционально `publish`.

Правила:

- стабильный `id`: `[a-z0-9_-]{1,32}`;
- установка компонента через `netductor install` **регистрирует** (или сидирует) запись;
- удаление компонента — soft-disable записи;
- `kind=egress` — общий выход в интернет, не VIP.

**Service-net:** предпочтительно `10.88.0.0/24` (пример) на primary. Пока сети нет — допустим interim через loopback + route только для разрешённых user; это должно быть явно в комментариях кода.

### 3.2 Объект политики

Общая форма для пользователей и роутеров: `subject_type`, `subject_id`, `allow_internet`, `services[]`, `services_mode` (`list`|`all`), метаданные обновления.

- `services_mode=all` — все internal из каталога;
- новый human user: internet да, internal пусто;
- новый edge после approve: пресет сайта / edge-default (internet через secondary + soft WAN fallback), internal по opt-in.

### 3.3 Пресеты

| id | Смысл |
|----|--------|
| `full` | internet + все internal |
| `media` | lampac (+ позже nvr) |
| `dev` | git + registry |
| `edge-default` | uplink VLESS + soft WAN; без admin-сервисов |

---

## 4. Enforcement

Клиенту нельзя доверять. Режет **primary**.

1. UUID/имя VLESS → `subject_id`.  
2. Route: internal/service-net — только если сервис в policy; иначе **reject**.  
3. `allow_internet=false` — режет обычный egress; **не** ломать SP/PS и служебные пути.  
4. Роутеры — отдельный peer `edge-<id>`, тот же движок route.  
5. Публикация приложений: VIP service-net (предпочтительно) или interim loopback.  
6. Опционально nft как второй рубеж (v1.1).

---

## 5. API

Те же эндпоинты и CLI, что в EN-документе (`/api/services`, policy GET/PUT для vpn user и edge, `policy/apply`, CLI `services` / `vpn policy` / `edge policy`). Всё в opcatalog. Apply идемпотентный.

---

## 6. UI

- **Пользователи** — только люди/устройства-клиенты.  
- **Роутеры** — отдельная Политика с тем же каталогом галочек.  
- TG — по шаблонам TG-UI; галочки политики обязательны; CRUD каталога можно оставить Web/TUI/CLI.  
- Паритет Web / TUI / CLI / TG по редактированию policy.

---

## 7. OpenWrt

| Тема | Правило |
|------|---------|
| Идентичность | Отдельный VLESS peer, скрыт из Users |
| Трафик дома | Private Wi‑Fi → VLESS secondary + soft WAN fallback |
| Internal | Opt-in галочки |
| Шаблоны сети | Остаются; policy — отдельный объект |
| LuCI / guest | Не смешивать с каталогом сервисов |

---

## 8. Фазы (чеклист)

**P0** модель данных и тесты → **P1** enforcement sing-box → **P2** API/CLI → **P3** UI всех поверхностей → **P4** пресеты, doctor, доки, релиз.

UI-галочки **не** выкатывать без P1.

---

## 9. Безопасность

Policy — только оператор. Каталог с произвольными endpoint — валидация private/service ranges. Internal не слушать на `0.0.0.0` «для удобства». Denied → reject, не silent open.

---

## 10. Критерии приёмки

1. Снять галочку lampac у `appletv` — нет доступа; у `operator` есть.  
2. Политика роутера не в разделе Users.  
3. Новый сервис в каталоге → новая галочка без правки каждого экрана.  
4. PUT policy / `vpn apply` пересобирает route без обязательного reboot.  
5. EN/RU UI и доки совпадают по смыслу.

---

## 11. Handoff

Начинать с **P0**. Peer’ы edge не в Users. После фазы — отметить чеклист и обновить `AGENT_HANDOFF.md`.
