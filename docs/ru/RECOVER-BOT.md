# Recover через бота / CLI (оператор)

Пример (актуальный тег подставляйте свой):

```bash
netductor recover --from-secondary https://SECONDARY:8790 \
  --recovery-token "$TOKEN" --key "$BACKUP_KEY"
```

Токен: `/etc/netductor/secrets/recovery_token` на secondary (или build-host recovery-arm).  
Ключ: offline `BACKUP_KEY` / `NETDUCTOR_BACKUP_KEY`.

После recover: `doctor`, vpn apply, SP/PS, heartbeat secondary.

Полный сценарий: `DISASTER-RECOVERY.md`. Не опирайтесь на устаревшие теги вроде v0.9.113 из старых примеров.
