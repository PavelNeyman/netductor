# Алерты → отдельный канал Telegram

Канонический маршрут алертов (приватные TG topics убраны).

## Настройка

1. Создайте канал (например `Netductor Alerts`).
2. Добавьте бота **админом** (право писать).
3. Узнайте id канала (перешлите сообщение в `@userinfobot` / `@getidsbot`, или Bot API `getUpdates` после поста).
4. На primary:

```bash
# secret-файл или conf — отрицательный id вида -100xxxxxxxxxx
echo -1001234567890 > /etc/netductor/secrets/telegram_alerts_chat_id
chmod 600 /etc/netductor/secrets/telegram_alerts_chat_id
systemctl restart netductor-telegram-bot
```

## Поведение

| Куда | Содержимое |
|--|--|
| **Канал** (`telegram_alerts_chat_id`) | batched alerts / warnings / service / updates |
| **ЛС оператора** (`telegram_admin_id`) | `/menu`, карточки VPN, QR, интерактивные tools |

- Посты в канал **без** `message_thread_id` (id топика ≠ id канала).
- Компактный re-pin хаба («меню внизу») **пропускается**, если задан alerts-канал — меню только в чате оператора.
- Карточки пользователей / QR остаются в ЛС (в канал не выталкиваются).

## Режим topics (legacy)

Старые topic-кнопки перенаправляются на alerts-chat, если канал настроен.

## UI / CLI

```bash
netductor tg-alerts status
netductor tg-alerts set -100xxxxxxxxxx
netductor tg-alerts test
netductor tg-alerts clear
```

- API: `GET/POST /api/telegram/alerts-chat` с session (`{"chat_id":"-100…"}` или `{"clear":true}`)
- TG: экран Topics → **Channel** / `/alerts_chat -100…`
- op TUI: Telegram install → **TG alerts channel id**

## Интерактив (Telegram)

Topics → **Channel** → **Set**:

1. Бот — **admin канала**.
2. **Переслать** любой пост канала в ЛС бота (рекомендуется) **или** ввести `-100…`.
3. Либо кнопка **known chat** (бот помнит каналы из `my_chat_member` / постов / forwards).

Боты **не** могут перечислить все каналы, где они admin — только чаты, по которым были updates.
