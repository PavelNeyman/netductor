# Backups

## Contents
Archive packs (when present on disk):

- `/etc/netductor` — secrets, VPN users, conf
- `/etc/blocky` — DNS config
- `/etc/sing-box` — certs/config if any
- `/var/lib/netductor` — state, sites, nodes, quotas
- `/opt/netductor/lampac` — app data (image re-pulled)
- `/opt/netductor/profiles` — e.g. nd-oc.conf

Sidecars next to archives: `COMPONENTS.txt`, recovery key note.

## UI
Tools → Backup: schedule, Run now, List + Restore, Keep N.

## CLI
`netductor backup` / `netductor restore [--key KEY] file.ndenc`
