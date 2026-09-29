# Recover Telegram bot without the bot

Config backups do **not** restore a dead `netductor-tg` process. Use SSH + GitHub assets.

```bash
TAG=v0.9.113   # or latest
curl -fsSL -o /tmp/netductor-linux-amd64   "https://github.com/PavelNeyman/netductor/releases/download/${TAG}/netductor-linux-amd64"
curl -fsSL -o /tmp/netductor-tg-linux-amd64   "https://github.com/PavelNeyman/netductor/releases/download/${TAG}/netductor-tg-linux-amd64"

systemctl stop netductor-telegram-bot netductor-api
install -m 755 /tmp/netductor-linux-amd64 /usr/local/bin/netductor
install -m 755 /tmp/netductor-tg-linux-amd64 /usr/local/bin/netductor-tg
systemctl start netductor-api netductor-telegram-bot

systemctl is-active netductor-api netductor-telegram-bot
journalctl -u netductor-telegram-bot -n 40 --no-pager
/usr/local/bin/netductor-tg -h 2>&1 | head -3
netductor version
```

Topics: after bot is up, Topics → Reconcile (or wait ≤6h). Clearing Telegram chat history deletes topics client-side; reconcile recreates bootstrap from `topics-backup.json` names.


## Memory guard (0.9.115+)

Unit has `MemoryMax=512M`. If bot is OOM-killed, check backup size and that update is not running inside the bot process.
