# Alerts → separate Telegram channel

Preferred over forum **topics** (topics are awkward for operators).

## Setup

1. Create a channel (e.g. `Netductor Alerts`).
2. Add the bot as **admin** (post messages).
3. Get channel id (forward a message to `@userinfobot` / `@getidsbot`, or use Bot API `getUpdates` after posting).
4. On primary:

```bash
# secret file or conf — negative id like -100xxxxxxxxxx
echo -1001234567890 > /etc/netductor/secrets/telegram_alerts_chat_id
chmod 600 /etc/netductor/secrets/telegram_alerts_chat_id
systemctl restart netductor-telegram-bot
```

## Behaviour

| Destination | Content |
|--|--|
| **Channel** (`telegram_alerts_chat_id`) | batched alerts / warnings / service / updates |
| **Operator DM** (`telegram_admin_id`) | `/menu`, VPN user cards, QR, interactive tools |

- Channel posts do **not** use `message_thread_id` (topic ids are not channel ids).
- Compact hub re-pin (“Menu stays at the bottom”) is **skipped** when the alerts channel is set — menu lives only in the operator chat.
- User cards / QR stay in the operator chat (never forced into the channel).

## Topics mode (legacy)

If `telegram_alerts_chat_id` is empty, routing falls back to forum topics in `topics.json` (see [TG-ALERTS-TOPIC.md](TG-ALERTS-TOPIC.md)).
