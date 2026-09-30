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
