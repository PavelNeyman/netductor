# Runbook — day 1 (Netductor)

## 1. Install on Debian VPS
```bash
curl -fsSL https://raw.githubusercontent.com/PavelNeyman/netductor/main/bootstrap.sh | bash
netductor install
netductor doctor
```

## 2. Session for Admin
```bash
netductor vpn session 72
# SSH tunnel → http://127.0.0.1:8787/admin/
```

## 3. Telegram
Secrets: `telegram_bot_token`, `telegram_admin_id`.  
`systemctl enable --now netductor-telegram-bot`.

## 4. VPN link
```bash
netductor vpn link operator vless
netductor vpn link operator sub
netductor vpn refresh-links
```
Fragments: `#nd-secondary` / `#nd-core`. Rename keeps UUID.

## 5. Secondary + backup peer
```bash
netductor secondary provision  # alias: relay provision --host IP --user root --password '…'
netductor backup peer-set 'root@SECONDARY:/var/lib/netductor/backups/peers/core/'
```

## 6. Checks
```bash
netductor doctor
netductor vpn apply --dry-run
netductor vpn mismatch
netductor audit tail
```

Carrier WL needs an **L3-whitelisted** RU IP. Default SNI `api.vk.me`. See [WL.md](WL.md).
