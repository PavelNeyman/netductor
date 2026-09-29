# Optional: alerts in a private topic

Bot API supports **topics in private chats** with the bot. Alerts can go to one topic; menu stays in General.

## Setup

1. In Telegram, open the chat with the netductor bot.
2. Enable **Topics** for this chat (client UI — private chat topics).
3. Create a topic, e.g. **Alerts**.
4. Send any message **inside that topic**.
5. On the VPS, read the thread id from bot updates:

```bash
TOKEN=$(cat /etc/netductor/secrets/telegram_bot_token)
curl -sS "https://api.telegram.org/bot${TOKEN}/getUpdates" | python3 -m json.tool | less
# look for "message_thread_id": <number>
```

6. Save it:

```bash
echo -n '123456' > /etc/netductor/secrets/telegram_alerts_thread_id
chmod 600 /etc/netductor/secrets/telegram_alerts_thread_id
systemctl restart netductor-api   # notify runs from node; tg uses same secrets dir
# or restart both api + telegram-bot
systemctl restart netductor-telegram-bot
```

If the file is missing, alerts go to the normal private chat (as before).
