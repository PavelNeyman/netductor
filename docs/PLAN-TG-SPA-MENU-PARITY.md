# TG SPA menu parity

Official widget profile (**core.telegram.org/bots/widgets**): **no keyboard / no input/textarea/select** — only `<button>`. Free text is impossible inside the widget.

## Input alternative: on-screen pad

Compose draft in bot state; pad keys send `p:<char>|p:bs|p:cl|p:ok` (≤64 bytes).

| Mode | Entry | OK does |
|------|-------|---------|
| quota | User card → Custom GiB | `SetSoftLimitGB` |
| rename | User card → Rename | `vpn rename` |
| add | Users → Add | `vpn add` |

Hybrid chat message still works if pad is awkward on small screens.

## Wizards in SPA

| Area | Coverage |
|------|----------|
| Alerts | status, **# pick known chat**, test, clear |
| Git | project list, **# → sync/ship** |
| Edge guest | list, **# → grant/revoke** |
| Edge LuCI | status, **on 4h / off** |
| Stack | heal, rollback confirm, watchdog |
| Reboot | pick + challenge codes |
| Backup | # → restore confirm |
| User | enable/disable/link/sub/quota presets+custom/rename/revoke |

## Still out of widget (by platform)

- QR **photo** (img data: possible but base64 blows 1 KiB state; link/sub text stays; photo = classic send)
- Git add project / migrate (needs long URLs — pad or classic)
- Edge guest multi-step facts/enroll beyond grant/revoke
