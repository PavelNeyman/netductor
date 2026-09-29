# Alert topics (manual)

**No auto-create.** You create topics in Telegram; the bot only stores `message_thread_id`.

## Prerequisites

1. @BotFather → your bot → **Threaded Mode** → Enable  
2. Open private chat with the bot → create topics (e.g. Alerts, Warnings, Service, Updates)

## Assign a topic

**From inside the topic** send:

```text
/topic alerts
/topic warnings
/topic service
/topic updates
```

Or: Operator → **📁 Topics** (status table).

The bot reads `message_thread_id` from that message and saves  
`/var/lib/netductor/tg/topics.json`.

## Routing

| Topic role | Typical alert keys |
|------------|-------------------|
| alerts | default / offline |
| warnings | mismatch, quota, warn |
| service | backup, mtls, svc-paths |
| updates | release, update |

## Legacy override

If `/etc/netductor/secrets/telegram_alerts_thread_id` is set, **all** alerts go to that single thread (ignores the table). Remove the file to use per-role topics.
