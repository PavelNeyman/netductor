# Edge после переустановки primary

См. также английскую версию: [EDGE-REINSTALL.md](../EDGE-REINSTALL.md).

## Суть

Реестр edge и токены живут на **primary**. Чистая переустановка без бэкапа сбрасывает доверие. Старый `device_token` на роутере **не** восстанавливает approved автоматически.

## Рекомендуемые пути

1. **Restore бэкапа** primary (edge + secrets) — роутеры продолжают работать.
2. **LAN recovery**: `edge recovery` → на Wi‑Fi точки `http://<router>:7879/netductor-recovery` → код → **pending** → Approve.  
   `CONTROL_ONLY=1` — настройки Wi‑Fi/UCI **не** перекатываются.
3. Pre-declare: `edge register <device_id> --site …`.

Firewall: порт **7879 только в LAN**.
