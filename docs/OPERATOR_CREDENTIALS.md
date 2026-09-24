# Operator credentials file

After **primary** or **secondary** deploy, netductor writes:

`~/.netductor/credentials/<role>-<host>-<timestamp>.txt` (mode `0600`)

## Contents

| Field | Purpose |
|-------|---------|
| SSH command + key path | Daily access (password auth is off after harden) |
| `BACKUP_KEY` | Decrypt `.ndenc` backups / `recover --key` |
| `RECOVERY_TOKEN` | Download backup from secondary after `recovery arm` (often only on secondary) |

## Recovery flow (no port-knock)

1. SSH to secondary: `netductor recovery arm --ttl 30m`
2. On new primary: `netductor recover --from-secondary http://SECONDARY:8790 --recovery-token … --key …`
3. On secondary: `netductor recovery disarm`

## RU

Файл создаётся на **вашем Mac** после деплоя. Сохраните в менеджер паролей.  
Приватный ключ SSH на сервер не копируется. Пароль root после harden не работает.
