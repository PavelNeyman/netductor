# Runbook — day 1 (Netductor)

Baseline: **v0.8.25**. Full dual-node narrative: [DEPLOY.md](DEPLOY.md) · [FLEET.md](FLEET.md).

## 1. Install on Debian VPS
```bash
export NETDUCTOR_VERSION=0.8.25   # bootstrap default is 0.8.1; pin explicitly on reinstall
curl -fsSL https://raw.githubusercontent.com/PavelNeyman/netductor/main/bootstrap.sh | bash
# Or binary from https://github.com/PavelNeyman/netductor/releases/tag/v0.8.25

mkdir -p /etc/netductor/secrets
# echo BOT_TOKEN > /etc/netductor/secrets/telegram_bot_token
# echo TG_USER_ID > /etc/netductor/secrets/telegram_admin_id
# chmod 600 /etc/netductor/secrets/*

netductor install
netductor doctor
```

After `install`, **SSH password auth is disabled**. Save `/root/.ssh/id_ed25519` via provider console. See [SSH.md](SSH.md).

## 2. Session for Admin
```bash
netductor vpn session 72
# SSH tunnel → http://127.0.0.1:8787/admin/
```
Admin stays VPN/localhost-only by policy — [ADMIN.md](ADMIN.md).

## 3. Telegram
Secrets: `telegram_bot_token`, `telegram_admin_id`.  
`systemctl enable --now netductor-telegram-bot` (usually enabled by install).

## 4. VPN link
```bash
netductor vpn link operator vless
netductor vpn link operator hy2
netductor vpn refresh-links
```
Subscription feature removed. Client notes: [SHADOWROCKET.md](SHADOWROCKET.md) · [VPN-USERS.md](VPN-USERS.md).

## 5. Secondary + backup peer
```bash
# Preferred (from primary):
netductor fleet provision-secondary --host RU_IP --password '…' [--sni api.vk.me]
netductor secondary sync
netductor fleet status
netductor backup peer-set 'root@SECONDARY:/var/lib/netductor/backups/peers/core/'
```
Secondary = **VPN entry only** — [PLAN-SECONDARY-VPN-ONLY.md](PLAN-SECONDARY-VPN-ONLY.md).

## 6. Checks
```bash
netductor doctor
netductor vpn apply --dry-run
netductor vpn mismatch
netductor audit tail
netductor fleet status
```

Carrier WL needs an **L3-whitelisted** RU IP. Default SNI often `api.vk.me`. See [WL.md](WL.md) · [SNI-PRESETS.md](SNI-PRESETS.md).
