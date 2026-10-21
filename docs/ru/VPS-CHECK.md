# Проверка VPS (pre-check)

Реализация scorecard на Go (без `curl | bash` сторонних скриптов).

Пакеты на цели: `curl`, `iperf3`, `sysbench`. Профили: `secondary` (РФ entry), `primary` (зарубежный), `generic`.

```bash
netductor-op vps-check --host IP --password '…' --profile secondary --primary PRIMARY_IP
netductor vps-check --profile secondary --json   # day-2 на ноде
```

Web: Deploy → VPS check / Secondary → pre-check. TUI: мастер VPS pre-check; у Secondary поле precheck.

Job: `vps.precheck`. Закладки хостера — отдельно: `netductor host-audit`.
