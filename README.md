# Netductor

Self-hosted **network control plane** (Debian primary + RU **secondary** VPN entry + OpenWrt edge):

- VPN: VLESS+Reality (sing-box)
- DNS: Blocky (localhost)
- Admin API + web UI (localhost / via VPN)
- Telegram operator bot
- Edge agent (OpenWrt), secondary agent (RU)
- Thin git + local OCI registry + optional Lampac
- Encrypted backups + agent offsite pull to RU + recovery API

**Current release:** **v0.9.71**

```bash
# Mac
brew install netductor   # or download from Releases
netductor version        # 0.9.71

# Full deploy from workstation
netductor deploy primary --host … --password … --key ~/.ssh/netductor_primary \
  --sni api.vk.me --tg-token … --tg-admin … --with-lampac
export NETDUCTOR_SSH_PORT=52222
netductor deploy secondary --primary … --primary-key … --host … --password … --sni api.vk.me
```

**Docs:** [index](docs/README.md) · [DEPLOY-MAC](docs/DEPLOY-MAC.md) · [FLEET](docs/FLEET.md) · [BACKUP](docs/BACKUP.md) · [PORTS](docs/PORTS.md) · [ARCHITECTURE](docs/ARCHITECTURE.md) · [AGENT_HANDOFF](docs/AGENT_HANDOFF.md) · [RECOVER-DRILL](docs/RECOVER-DRILL.md) · [AGENTS.md](AGENTS.md)

RU: [docs/ru/](docs/ru/)

### Useful commands
```bash
netductor doctor
netductor fleet status
netductor secondary status
netductor vpn list
netductor backup
netductor recover --from-secondary https://SECONDARY:8790 --recovery-token TOKEN
netductor git list
netductor registry status
netductor tui
```
