# Smoke

```bash
netductor doctor
netductor vpn list
netductor vpn link operator
```

# Smoke-чеклист (русский)

**EN:** [../SMOKE.md](../SMOKE.md)

1. Установка завершилась с кодом 0; есть `/etc/netductor/READY.txt`.
2. `systemctl is-active sing-box blocky` → active.
3. `netductor vpn list` показывает `operator`; в `subscription.txt` есть `vless://` и `hysteria2://`.
4. `sing-box check -c /usr/local/etc/sing-box/config.json` OK.
5. Слушают порты 443, 8443, 53.
6. `dig @127.0.0.1 example.com` работает.
7. Импорт subscription на телефоне; трафик идёт.
8. `netductor vpn add smoke1 test && netductor vpn revoke smoke1`.
9. API: `netductor vpn session 1` + curl на `127.0.0.1:8787`.
10. Telegram `/status` `/ready` (если бот включён).
11. Панели через SSH-туннель; UFW в порядке; backup-скрипт на месте.
12. После reboot сервисы поднимаются.

Автоматический срез на сервере: `sudo netductor doctor` / `netductor doctor`.
