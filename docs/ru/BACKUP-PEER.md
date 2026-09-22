# Backup peer (устаревший SSH)

**Устарело.** Primary **не** копирует бэкапы на secondary через SCP.

Актуальная модель: [BACKUP.md](../BACKUP.md) / [ru — см. BACKUP.md](../BACKUP.md)

- После `netductor backup` secondary-агенты получают **`backup_pull`** (HTTPS mTLS).
- Файлы: `/var/lib/netductor/backups/peers/core/`
- DR: API secondary **:8790** + `netductor recover --from-secondary`

`backup peer-set` (SCP) не используется при автодеплое.
