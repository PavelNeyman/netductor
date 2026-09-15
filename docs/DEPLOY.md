# Deploy & test netductor

Full dual-node narrative: [FLEET.md](FLEET.md) · [AGENT_HANDOFF.md](AGENT_HANDOFF.md)

## 1. Requirements

- Debian 12/13 VPS (root SSH), public IPv4
- Optional second RU VPS for secondary (whitelist entry)
- Optional: Telegram bot token + numeric admin user id

## 2. Primary bootstrap

After `install`, **SSH password auth is disabled**. Private key: `/root/.ssh/id_ed25519` (copy via provider console if needed). Secondary gets the same pubkey during `provision-secondary`.


```bash
wget -qO /usr/local/bin/netductor \
  https://github.com/PavelNeyman/netductor/releases/download/v0.7.0-dev/netductor-linux-amd64
chmod 755 /usr/local/bin/netductor
netductor version

mkdir -p /etc/netductor/secrets
echo 'BOT_TOKEN' > /etc/netductor/secrets/telegram_bot_token
echo 'YOUR_TG_USER_ID' > /etc/netductor/secrets/telegram_admin_id
chmod 600 /etc/netductor/secrets/*

netductor install
netductor doctor
```

Reality SNI (WL-oriented default in ops): **`api.vk.me`**

```bash
netductor vpn set-sni api.vk.me
```

## 3. Secondary (from primary)

```bash
netductor fleet provision-secondary --host RU_IP --password '…' --sni api.vk.me
netductor fleet status
netductor vpn refresh-links
```

Client links should show **secondary IP** when secondary is online.

## 4. Verify

```bash
netductor doctor
systemctl is-active sing-box blocky netductor-api netductor-telegram-bot
netductor vpn list
netductor fleet status
curl -fsS http://127.0.0.1:8788/api/bot-status
```

## 5. VPN client

```bash
netductor vpn link operator vless
netductor vpn link operator hy2
```

Import into Shadowrocket / Happ / v2rayN / sing-box. Subscription feature **removed**.

## 6. Telegram

Admin id must match secrets. `/menu` — users → link/QR (VLESS default, toggle HY2).

## 7. Admin SPA

```bash
netductor vpn session 72
ssh -L 8787:127.0.0.1:8787 root@PRIMARY
```

Open `http://127.0.0.1:8787/admin/`

## 8. Backup / recover

```bash
netductor backup
# keep BACKUP_KEY.txt + COMPONENTS.txt with the .ndenc

netductor recover --key "$KEY" /path/to/file.ndenc
```

Cross-peer: `netductor backup peer-set root@SECONDARY:/var/lib/netductor/backups/peers/core/`

## 9. Update binary

```bash
systemctl stop netductor-api netductor-telegram-bot
wget -qO /usr/local/bin/netductor https://github.com/PavelNeyman/netductor/releases/download/v0.7.0-dev/netductor-linux-amd64
wget -qO /opt/netductor/bin/netductor-tg https://github.com/PavelNeyman/netductor/releases/download/v0.7.0-dev/netductor-tg-linux-amd64
chmod 755 /usr/local/bin/netductor /opt/netductor/bin/netductor-tg
systemctl start netductor-api netductor-telegram-bot
```

## 10. Uninstall

```bash
netductor uninstall   # keep configs
netductor purge       # wipe configs/data
```
