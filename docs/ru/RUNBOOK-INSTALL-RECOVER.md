# Runbook: install и recover

**RU** · [EN](../RUNBOOK-INSTALL-RECOVER.md)

## Install / update
Идемпотентно: `netductor install …`, `netductor update` с Release → `/usr/local/bin`.  
FHS: conf в `/etc/netductor`, state в `/var/lib/netductor`.

## Recover (unattended, 0.9.71+)
1. Бинарь с Release на чистый VPS  
2. Secondary: `recovery arm`  
3. `recover --from-secondary https://SEC:8790 --recovery-token … --key …`  
4. Early pubkey → harden → install COMPONENTS → tar → **post-restore pass** → ufw secondary → done  

vpn-users до restore может soft-fail → `(continuing)`.

Критерий: doctor fail=0, bot/api/sing-box active, secondary online, **без** ручного scp.

## После recover проверить
- LE/redirect: если нет `/etc/letsencrypt` — `tls le`  
- secondary sing-box: geoip download через **direct**, не Reality uplink (0.9.72)  
- `ufw status` :8789 для IP secondary  

См. [RECOVER-DRILL](RECOVER-DRILL.md).
