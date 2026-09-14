# Runbook — день 1 (Netductor)

## 1. Установка на Debian VPS
```bash
curl -fsSL https://raw.githubusercontent.com/PavelNeyman/netductor/main/bootstrap.sh | bash
netductor install
netductor doctor
```

## 2. Session для Admin
```bash
netductor vpn session 72   # токен → SSH-туннель → http://127.0.0.1:8787/admin/
```

## 3. Telegram
Секреты: `telegram_bot_token`, `telegram_admin_id`.  
Сервис: `systemctl enable --now netductor-telegram-bot`.

## 4. Ссылка VPN
```bash
netductor vpn link operator vless      # primary (#nd-relay если online)
netductor vpn link operator sub        # subscription без HY2
netductor vpn refresh-links            # пересобрать все артефакты
```
Fragment: `#nd-relay` / `#nd-core` — не имя пользователя. Rename не ломает UUID.

## 5. Relay (RU entry)
```bash
netductor relay provision --host IP --user root --password '…'
netductor backup peer-set 'root@RELAY:/var/lib/netductor/backups/peers/core/'
```

## 6. Проверки
```bash
netductor doctor
netductor vpn apply --dry-run
netductor vpn mismatch
netductor audit tail
```

Под БС: нужен **L3-проходимый** RU IP; SNI по умолчанию `api.vk.me`. См. [WL.md](WL.md).
