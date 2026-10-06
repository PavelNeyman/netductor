# Cleanup (upgrade leftovers)

```bash
netductor cleanup          # dry-run + size estimate
netductor cleanup --apply  # remove
netductor cleanup --json
```

Removes: `stack/attempt`, `/tmp/nd-sb-*`, `config.json.tmp`, legacy test WG/units (backbone/awg/strongswan).

**Never removes:** `stack/prev`, `backups/`, `baseline/`, live sing-box config, service WG (`nd-svc-sp` / `nd-svc-ps`).

After successful `stack apply`, cleanup runs automatically (apply mode).

API: `GET/POST /api/cleanup` (session). TG Tools: Cleanup / Apply.
