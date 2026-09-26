# TG UI pattern (locked after DNS 0.9.28)

## Rules

1. **Table** = overview only (`#`, name, status icons). **No** `<tg-button>` inside `<td>` (Telegram does not deliver callbacks).
2. **Actions** = `<tg-button-row>` under the table. Prefer **numbers** matching `#` when many items; several buttons per row (≤8).
3. **Keyboard under message** = navigation only (`Back` → parent, `Main`).
4. **Refresh** = `reply()` → edit rich first; delete+send only if edit fails.
5. **Style**: `link` = compact toggle; `success`/`danger`/`primary` for strong actions; short labels / emoji OK.

## Applied

| Screen | Pattern |
|--------|---------|
| DNS lists | numbered toggles + 🔄 |
| Backup list | numbered restore |
| Backup schedule | time / ▶ / 📋 / N in body |
| Pending edge | ✅n 🚫n in body |
| Locations list | numbered open |
| Location card | ✏️ 🗑 in body |
| Users / VPN | already row-buttons (reference) |

## Still on keyboard (debt)

None material for day-2 hubs (0.9.38). Product pickers that need free-text still use wait-state messages.


## 0.9.30

- Users list: table + numbered open (green=on)
- User hub: compact emoji row
- Nodes list: numbered open; node card emoji ops
- Git / Registry / SSH hosts: same pattern


## 0.9.32

- NVR hub: table + emoji body actions
- NVR cameras: table + P/R/S/PTZ rows per cam
- NVR sites: numbered edge lease pick
- CDN-XHTTP: idea only in OPEN_ITEMS


## 0.9.37–0.9.38

- Catalog section/result: body actions + 📄 JSON
- SSH hosts: numbered forget + clear in body
