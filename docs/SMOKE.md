# Smoke

```bash
netductor doctor
netductor vpn list
netductor vpn link operator
```

# Smoke-test checklist (English)

**RU:** [ru/SMOKE.md](ru/SMOKE.md)

1. Install exits 0; `/etc/netductor/READY.txt` present.
2. `systemctl is-active sing-box blocky` → active.
3. `netductor vpn list` shows `operator`; `subscription.txt` has `vless://` and `hysteria2://`.
4. `sing-box check -c /usr/local/etc/sing-box/config.json` OK.
5. Listeners on 443, 8443, 53.
6. `dig @127.0.0.1 example.com` works.
7. Import subscription on a phone; traffic works.
8. `netductor vpn add smoke1 test && netductor vpn revoke smoke1`.
9. API: `netductor vpn session 1` + curl `127.0.0.1:8787`.
10. Telegram `/status` `/ready` (if bot enabled).
11. Panels via SSH tunnel; UFW sane; backup script present.
12. Reboot; services return.

Automated host slice: `sudo netductor doctor` / `netductor doctor`.
