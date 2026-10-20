# Disaster recovery (primary недоступен)

1. Доступ к **secondary** (SSH порт 52222, только ключ после harden).
2. Восстановить данные primary из peer-копии:
   ```bash
   netductor recover --from-secondary https://SECONDARY:8790 \
     --recovery-token "$TOKEN" --key "$BACKUP_KEY"
   ```
3. `netductor doctor`, при необходимости LE, `vpn` apply, проверить SP/PS.
4. Heartbeat / uplink secondary agent.
5. `netductor update apply`, если версии разъехались.

Шифрованные бэкапы попадают на secondary через agent `backup_pull`. Ключ offline (`--key` / `NETDUCTOR_BACKUP_KEY`).

TG: **Menu → 📡 Fleet → DR** checklist.

## Хранение ключа (принято)

Offsite-копии в `peers/core/` на secondary могут лежать рядом с `BACKUP_KEY.txt` (mode 0600) у `.ndenc`. Шифрование «на диске» secondary не даёт второго фактора: кто читает архив, читает и ключ. Так задумано, чтобы DR не зависел от offline-ключа у оператора. Не считайте диск secondary вторым фактором.
