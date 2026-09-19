# Upgrade

## Primary (VPS)

1. **TG → Tools → Updates → Update primary** — downloads GitHub Release assets (`netductor`, `netductor-tg`), writes `/etc/netductor/VERSION`, restarts `netductor-api` + `netductor-telegram-bot`.
2. Or manual:

```bash
TAG=v0.8.0
curl -fsSL -o /usr/local/bin/netductor \
  "https://github.com/PavelNeyman/netductor/releases/download/${TAG}/netductor-linux-amd64"
chmod 755 /usr/local/bin/netductor
# same for netductor-tg → /opt/netductor/bin/
systemctl restart netductor-api netductor-telegram-bot
echo 0.8.0 > /etc/netductor/VERSION
```

## Secondary

Via `netductor secondary` / Nodes UI upgrade commands.

## OpenWrt agents

**No auto-rollout.** Operator runs `edge cmd <id> agent_update` with release binary URL|sha256.  
Shipping a new primary does **not** force-update all agents.

## Policy

| Component | Source | Who decides |
|-----------|--------|-------------|
| primary bin + tg | GitHub Release | operator (TG/CLI) |
| secondary | cmd from primary | operator |
| edge agent | `agent_update` | operator, per device |
| `main` branch | may lead release | not for prod auto |
