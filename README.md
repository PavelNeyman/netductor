**New agent / new chat:** start with [docs/AGENT_HANDOFF.md](docs/AGENT_HANDOFF.md) · [docs/TG-UI.md](docs/TG-UI.md) · [docs/FLEET.md](docs/FLEET.md) · [docs/OPEN_ITEMS.md](docs/OPEN_ITEMS.md).

# Netductor

Self-hosted **network control plane** for a Debian VPS (+ OpenWrt edge, optional MikroTik site, RU **secondary** VPN entry):

- VPN: VLESS+Reality, Hysteria2 (sing-box)
- DNS: Blocky
- Admin API + web UI (localhost / VPN)
- Telegram operator bot
- Edge agent (OpenWrt), secondary agent (RU VPS)
- SSH TOFU known_hosts management

```bash
# Mac: brew install netductor  OR  brew install --HEAD netductor
export NETDUCTOR_VERSION=0.8.19
curl -fsSL https://raw.githubusercontent.com/PavelNeyman/netductor/main/bootstrap.sh | bash
sudo netductor install
netductor tui
```

**Docs:** [INSTALL](docs/INSTALL.md) · [DEPLOY](docs/DEPLOY.md) · [FLEET](docs/FLEET.md) · [ARCHITECTURE](docs/ARCHITECTURE.md) · [ROADMAP](docs/ROADMAP.md) · [OPEN_ITEMS](docs/OPEN_ITEMS.md) · [TOFU](docs/TOFU.md) · [AGENTS.md](AGENTS.md)

RU: [docs/ru/](docs/ru/)

### Useful commands
```bash
netductor doctor
netductor vpn list
netductor vpn refresh-links
netductor audit tail
netductor vpn rename old new
netductor backup peer-set root@peer:/path/
netductor vpn mismatch
netductor ssh-hosts list
netductor fleet status
netductor secondary status
netductor secondary sync
```
