**[EN](../PLAN-TG-HTML-WIDGETS-MENU.md)** · RU

# План: меню TG-оператора на HTML Widgets (Bot API 10.4)

**Статус:** Phase 0+1 в работе (registry, status+channel; флаг off) · **Флаг:** `NETDUCTOR_TG_WIDGETS=1` (по умолчанию выкл. до soak)  
**База:** CF-18.4 status widget (`widget_status.go`)  
**Связано:** [TG-UI-PATTERN.md](../TG-UI-PATTERN.md) — rich A/B/C остаются fallback

## Цель

При включённом флаге экраны **операторского меню** открываются как **HTML-виджеты** в сообщении (live UI, `widget_query`, refresh без нового пузыря).  
При выключенном — **как сейчас** (rich + tg-button).

По умолчанию **выкл.**, пока не прогоним на боевом ЛС. Потом — default on.

## Не делаем

Mini App, виджеты в группах/каналах, TON, переписка алертов в канале, topics/guest.

## Ограничения 10.4

Только private, protected content, HTML ≲ 32 KiB, без внешнего fetch из виджета, клиент ≥ 13.2. При ошибке `sendWidget` — fallback на rich.

## Архитектура

`widgetsEnabled()` → registry экранов → `sendWidget` / `editMessageWidgetData` / `handleWidgetQuery`.  
Общий HTML-shell + `chat_data` JSON по экранам. Деструктивные действия — подтверждение **классическим** сообщением (как сейчас).

Главное меню предпочтительно оставить classic keyboard; виджеты — контентные экраны.

## Фазы

| Фаза | Содержание |
|------|------------|
| **0** | Вынести foundation из status: registry, apiPostWidget, тесты размера, gate |
| **1** | Status, Channel, Secondary status, Netmap, Probes/Metrics summary, Versions |
| **2** | Списки: Users, devices, Nodes, Fleet, Sites, SSH, Backup, Secondary, Pending |
| **3** | Карточки: User hub, Node, Operator, Addons, Svc-paths (confirm classic) |
| **4** | Остальной catalog по трафику |
| **5** | Soak → default on (`=0` чтобы выкл.) |

## Включение теста

```bash
mkdir -p /etc/systemd/system/netductor-telegram-bot.service.d
printf '%s\n' '[Service]' 'Environment=NETDUCTOR_TG_WIDGETS=1' \
  > /etc/systemd/system/netductor-telegram-bot.service.d/widgets.conf
systemctl daemon-reload && systemctl restart netductor-telegram-bot
```

## Критерий готовности

Фазы 1–3 под флагом, fallback rich, soak, default on, обновлён TG-UI-PATTERN.

Подробности и матрица экранов — в EN-версии.
