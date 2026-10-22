# Проверка VPS (pre-check)

Scorecard на Go без `curl | bash`. С 0.9.466: DNSBL/IP quality, geoblock-матрица, DPI (TLS SNI + DNS pollution), sysbench CPU/mem, fio/dd, iperf.

```bash
netductor-op vps-check --host IP --password '…' --profile secondary --primary PRIMARY_IP
netductor vps-check --profile secondary --json
```

Профили: `secondary` | `primary` | `generic`. Web/TUI/job `vps.precheck`. Закладки хостера: `netductor host-audit`.
