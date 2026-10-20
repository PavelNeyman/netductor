# План меню Telegram (реализовано)

Статус: **implemented** (0.9.288 / 0.9.289 + polish 0.9.319). Исторический план; живой хаб — код `cmd/netductor-tg`.

## Верхний хаб

| Callback | Экран |
|--|--|
| `m:status` | Status (+ channel / refresh / incident) |
| `m:users` | Users / VPN |
| `m:fleet` | Fleet |
| `m:tools` | Day-2 Tools (без home/users/fleet) |
| `m:operator` | Operator (Stack, Alerts, Session, Audit) |
| lang / help | Язык и справка |

Alerts только под Operator (`m:alerts-chat`), не в главном ряду и не в Tools.

## Чеклист callback map

- [x] Hub top-level: status / users / fleet / tools / operator
- [x] Alerts только под Operator
- [x] Tools фильтрует home/users/fleet (`toolsDay2Groups`)
- [x] `m:cat:home` / `m:ops:overview` → Status, не клон Tools
- [x] Status: firewall + channel one-liner; channel/digest/versions под Status где уместно
- [x] Operator: Stack + Alerts + Session + Audit
- [x] Неизвестный callback → hub fallback
- [x] EN/RU через `T()`

## Done when

- [x] После алерта один hub внизу (`repinHub` + batch)
- [x] Tools без дубля Status/Users
- [x] Чеклист по списку callback

### 0.9.289 nav

- Status keyboard: Channel + Refresh
- `m:digest` → fleet digest, parent Status
- Updates под Tools (`m:versions` / `m:updates`)

### 0.9.319 polish

- Help / Status / Users / canary через `T()`
- Tools = day-2 only

## Structure pass (2026-10-12)

| Проблема | Фикс |
|--|--|
| DR кнопка «Fleet» → `m:digest` | Label → `T("digest_btn")` |
| Хардкод EN на digest/DR | `T(...)` |
| `m:updates` и `m:versions` | Alias на один экран — ок |
| `topicsKeyboard` | Redirect на alerts chat (legacy) |

Чеклист inventory / map / back / i18n — закрыт.
