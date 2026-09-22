# Netductor

Self-hosted **network control plane** (Debian primary + RU **secondary** VPN entry + OpenWrt edge):

- VPN: VLESS+Reality, Hysteria2 (sing-box)
- DNS: Blocky (localhost)
- Admin API + web UI (localhost / via VPN)
- Telegram operator bot
- Edge agent (OpenWrt), secondary agent (RU)
- Thin git + local OCI registry + optional Lampac
- Encrypted backups + agent offsite pull to RU + recovery API

**Current release:** **v0.8.49**

```bash
# Mac
brew install netductor   # or download from Releases
netductor version        # 0.8.49

# Full deploy from workstation
netductor deploy primary --host … --password … --key ~/.ssh/netductor_primary \
  --sni api.vk.me --tg-token … --tg-admin … --with-lampac
export NETDUCTOR_SSH_PORT=52222
netductor deploy secondary --primary … --primary-key … --host … --password … --sni api.vk.me
```

**Docs:** [INSTALL](docs/INSTALL.md) · [DEPLOY](docs/DEPLOY.md) · [DEPLOY-WORKSTATION](docs/DEPLOY-WORKSTATION.md) · [FLEET](docs/FLEET.md) · [BACKUP](docs/BACKUP.md) · [PORTS](docs/PORTS.md) · [ARCHITECTURE](docs/ARCHITECTURE.md) · [AGENT_HANDOFF](docs/AGENT_HANDOFF.md) · [AGENTS.md](AGENTS.md)

RU: [docs/ru/](docs/ru/)

### Useful commands
```bash
netductor doctor
netductor fleet status
netductor secondary status
netductor vpn list
netductor backup
netductor recover --from-secondary http://SECONDARY:8790 --recovery-token TOKEN
netductor git list
netductor registry status
netductor tui
```
