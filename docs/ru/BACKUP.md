**RU** · [EN](../BACKUP.md)

# Backup и recover

## Состав бэкапа
Конфиги и данные сервисов (`/etc/netductor`, blocky, sing-box, `/var/lib/netductor`, profiles, lampac data…).  
**Бинарники не тащим** — ставятся с GitHub Release по списку COMPONENTS.

## Cross-backup
Агент secondary **pull** бэкапа primary по schedule (канон). Legacy scp primary→secondary **отключён**.

## Hostname
В бэкап пишется hostname; recover **не** генерит `nd-primary-<ip>` поверх — восстанавливает из backup. Свежий install — `applyHostname` / env.

## SSH ключи оператора
В бэкап только **public** keys (`operator_authorized_keys`). Private **никогда**.  
Recover: merge authorized_keys **до** harden из archive + `NETDUCTOR_OPERATOR_PUBKEY` / `_FILE`.

## Recovery security
- Secondary `:8790` только после `recovery arm` (SSH), TTL
- Отдаёт **шифрованный** `.ndenc` + COMPONENTS; ключ расшифровки **offline** (`--key` / env)
- Lockout после N неудачных auth
- Optional CIDR allow на recovery

```bash
ssh -p 52222 root@SECONDARY
netductor recovery arm --ttl 30m
# на чистом primary / Mac:
netductor recover --from-secondary https://SECONDARY:8790 \
  --recovery-token "$TOKEN" --key "$BACKUP_KEY"
netductor recovery disarm
```

Credentials после деплоя: `~/.netductor/credentials/` — [OPERATOR_CREDENTIALS.md](OPERATOR_CREDENTIALS.md).

## Post-restore (0.9.71+)
1. Pre-install COMPONENTS (vpn-users может soft-fail без secrets)
2. Распаковка archive
3. Post-restore pass (api/tg/backup/vpn-users)
4. **ensure-relay-uplink + vpn apply** (0.9.73) — conf = secrets, users не пустые
5. UFW secondary IPs, redirect LE

Без успешного `vpn apply` после recover **не считать VPN рабочим** — [RECOVER-DRILL.md](RECOVER-DRILL.md).


## Secondary без API primary

```bash
netductor backup secondary-local
netductor recovery arm
netductor backup push-recovery --url https://SEC:8790 --token … --file …
netductor backup push-ssh --host SEC --file …
```
