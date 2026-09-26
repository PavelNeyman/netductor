# Binary layout

**Canonical install root:** `/opt/netductor/`

| Path | Purpose |
|------|---------|
| `/opt/netductor/bin/netductor` | node CLI + API helpers |
| `/opt/netductor/bin/netductor-tg` | Telegram bot (**systemd ExecStart**) |
| `/opt/netductor/bin/netductor-agent` | edge agent (OpenWrt path differs) |
| `/etc/netductor/` | config |
| `/var/lib/netductor/` | state |

`/usr/local/bin/netductor*` may be a **symlink** to `/opt/netductor/bin/*` for PATH convenience — not a second copy of truth.

Deploy / self-update **must** write `/opt/netductor/bin/…` and restart the matching unit.
