# Deploy & test netductor

Baseline: **v0.8.1**. Prefer Release binaries: https://github.com/PavelNeyman/netductor/releases/tag/v0.8.1

Full dual-node narrative: [FLEET.md](FLEET.md) · [AGENT_HANDOFF.md](AGENT_HANDOFF.md) · [PLAN-SECONDARY-VPN-ONLY.md](PLAN-SECONDARY-VPN-ONLY.md)

## 1. Requirements

- Debian 12/13 VPS (root SSH), public IPv4
- Optional second RU VPS for secondary (whitelist entry)
- Optional: Telegram bot token + numeric admin user id

## 2. Primary bootstrap

After `install`, **SSH password auth is disabled**. Private key: `/root/.ssh/id_ed25519` (copy via provider console if needed). Secondary gets the same pubkey during `fleet provision-secondary`.

See [AGENT_HANDOFF.md](AGENT_HANDOFF.md) for dual-node order · [SSH.md](SSH.md).

```bash
export NETDUCTOR_VERSION=0.8.1
curl -fsSL https://raw.githubusercontent.com/PavelNeyman/netductor/main/bootstrap.sh | bash
# or:
# wget -qO /usr/local/bin/netductor \
#   https://github.com/PavelNeyman/netductor/releases/download/v0.8.1/netductor-linux-amd64
# chmod 755 /usr/local/bin/netductor

mkdir -p /etc/netductor/secrets
echo 'BOT_TOKEN' > /etc/netductor/secrets/telegram_bot_token
echo 'YOUR_TG_USER_ID' > /etc/netductor/secrets/telegram_admin_id
chmod 600 /etc/netductor/secrets/*

netductor install
netductor doctor
```

Reality SNI (WL-oriented default in ops): **`api.vk.me`** — [SNI-PRESETS.md](SNI-PRESETS.md)

```bash
netductor vpn set-sni api.vk.me
```

## 3. Secondary (from primary)

```bash
netductor fleet provision-secondary --host RU_IP --password '…' --sni api.vk.me
netductor secondary sync
netductor fleet status
netductor vpn refresh-links
```

Client links should show **secondary IP** when secondary is online. Operator CLI is `secondary` / `fleet` (not `relay`).

## 4. Verify

```bash
netductor doctor
systemctl is-active sing-box blocky netductor-api netductor-telegram-bot
netductor vpn list
netductor fleet status
```

## 5. VPN client

```bash
netductor vpn link operator vless
netductor vpn link operator hy2
```

Import into Shadowrocket / Happ / v2rayN / sing-box. Subscription feature **removed**. See [SHADOWROCKET.md](SHADOWROCKET.md).

## 6. Telegram

Admin id must match secrets. `/menu` — users → link/QR (VLESS default, toggle HY2). [TG-UI.md](TG-UI.md)

## 7. Admin SPA

```bash
netductor vpn session 72
ssh -L 8787:127.0.0.1:8787 root@PRIMARY
```

Open `http://127.0.0.1:8787/admin/` — VPN/localhost only by policy. [ADMIN.md](ADMIN.md)

## 8. Backup / recover

```bash
netductor backup
# keep BACKUP_KEY.txt + COMPONENTS.txt with the .ndenc

netductor recover --key "$KEY" /path/to/file.ndenc
```

Cross-peer: `netductor backup peer-set root@SECONDARY:/var/lib/netductor/backups/peers/core/` — [BACKUP.md](BACKUP.md)

## 9. Update binary

Prefer TG Tools → Updates (Release + SHA256SUMS). Manual:

```bash
systemctl stop netductor-api netductor-telegram-bot
wget -qO /usr/local/bin/netductor https://github.com/PavelNeyman/netductor/releases/download/v0.8.1/netductor-linux-amd64
wget -qO /opt/netductor/bin/netductor-tg https://github.com/PavelNeyman/netductor/releases/download/v0.8.1/netductor-tg-linux-amd64
chmod 755 /usr/local/bin/netductor /opt/netductor/bin/netductor-tg
systemctl start netductor-api netductor-telegram-bot
```

See [UPGRADE.md](UPGRADE.md).

## 10. Uninstall

```bash
netductor uninstall   # keep configs
netductor purge       # wipe configs/data
```

## Edge / NVR (after primary + secondary)

- OpenWrt agent: [EDGE-AGENT.md](EDGE-AGENT.md) · reinstall primary: [EDGE-REINSTALL.md](EDGE-REINSTALL.md)
- Cameras: [PLAN-NVR-TAPO.md](PLAN-NVR-TAPO.md)
