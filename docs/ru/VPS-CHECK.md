# Проверка VPS (pre-check)

Scorecard на Go без `curl | bash`. С 0.9.466: DNSBL/IP quality, geoblock-матрица, DPI (TLS SNI + DNS pollution), sysbench CPU/mem, fio/dd, iperf.

```bash
netductor-op vps-check --host IP --password '…' --profile secondary --primary PRIMARY_IP
netductor vps-check --profile secondary --json
```

Профили: `secondary` | `primary` | `generic`. Web/TUI/job `vps.precheck`. Закладки хостера: `netductor host-audit`.


## Security (0.9.467+)

Disk/fio tests use private `MkdirTemp` dirs (no fixed `/tmp` names). Password SSH requires `sshpass` or fails explicitly. DNSBL treats DNS errors as WARN, not clean.
