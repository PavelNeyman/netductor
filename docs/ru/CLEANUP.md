# Cleanup (остатки после upgrade)

```bash
netductor cleanup          # dry-run + оценка размера
netductor cleanup --apply  # удалить
netductor cleanup --json
```

Удаляет: `stack/attempt`, `/tmp/nd-sb-*`, `config.json.tmp`, legacy test WG/units (backbone/awg/strongswan).

**Никогда не трогает:** `stack/prev`, `backups/`, `baseline/`, живой конфиг sing-box, service WG (`nd-svc-sp` / `nd-svc-ps`).

После успешного `stack apply` cleanup выполняется автоматически (режим apply).

API: `GET/POST /api/cleanup` (session). TG Tools: Cleanup / Apply.
