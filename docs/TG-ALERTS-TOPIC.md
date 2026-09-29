# Alert topics

## Bootstrap (automatic)

With **Threaded Mode** enabled in @BotFather, the bot on first update creates exactly four topics and stores their ids in `…/tg/topics.json`:

| Topic | Role |
|-------|------|
| 🚨 Alerts | default / offline |
| ⚠️ Warnings | mismatch, quota |
| 🛠 Service | backup, mtls, svc-paths |
| 🔄 Updates | release / update |

No other topics are auto-created.

## Optional re-bind

From inside a topic: `/topic alerts` (or warnings / service / updates).

## Removed

`telegram_alerts_thread_id` secret is **not** used — routing is only via `topics.json`.
