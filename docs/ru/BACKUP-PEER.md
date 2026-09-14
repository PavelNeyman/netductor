# Перекрёстный бэкап core ↔ relay

Архив `/etc/netductor` (секреты, vpn-users, sni) обычно **несколько МБ** — места на обеих VPS хватает.

```bash
# на core: пуш на relay
mkdir -p setup on relay:
  ssh root@RELAY 'mkdir -p /var/lib/netductor/backups/peers/core'

netductor backup peer-set 'root@RELAY_IP:/var/lib/netductor/backups/peers/core/'
# ключ SSH без пароля (тот же, что для enroll)
netductor backup   # создаёт локальный + scp на peer

netductor backup peer-status
```

Обратно (на relay → core) — симметрично, другой target.

Таймер `netductor-backup.timer` уже вызывает `netductor backup` ежедневно; после `peer-set` offsite пойдёт сам.

Restore: `netductor restore /path/to/*.ndenc` (нужен `backup_key`).
