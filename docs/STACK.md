# Stack orchestrator (0.9.116+)

**EN** · Thin layer over netductor units + binaries (not a second control plane).

## Commands
```bash
netductor stack status
netductor stack apply v0.9.116   # or empty = latest
netductor stack rollback
netductor stack watchdog
```

## Behaviour
1. **apply** — snapshot current node+tg → `/var/lib/netductor/stack/prev/` → download from GitHub → restart api/bot → health; on fail → **rollback**
2. **status** — unit active + binary version
3. **watchdog** — try-restart failed core units (api/bot/sing-box/blocky)

TG/Web should call apply via API later; when bot is dead use SSH + `stack apply`.

## Not in scope
VPN business logic, configs in `/etc/netductor` (still backup/restore).

## Watchdog
```bash
netductor stack watchdog-install
```
Runs every 5 minutes; restarts failed core units only.


## TG
Tools → **Stack status** — table of units, Rollback, Watchdog.
Apply still via Updates (calls `stack apply` out-of-process).
Alerts: apply start/ok/fail, auto-rollback, watchdog restarts → Updates/Alerts topics.

## Apply flow
1. pre-backup (+ wait backup_pull ~20s)
2. snapshot prev binaries
3. download node+tg
4. restart api/bot
5. health → on fail auto-rollback + alert
