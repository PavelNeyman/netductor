# TG UI pattern (locked)

## Templates (only three)

### A — List
1. Title + optional one-line legend  
2. `<table>`: `#` | name | status (icons only, **no** buttons in cells)  
3. `<tg-button-row>`: numbers / short actions  
4. Keyboard: **Back** + **Main** only  

### B — Card
1. Title + legend (emoji · meaning)  
2. `<table>` field/value  
3. `<tg-button-row>` emoji ops  
4. Keyboard: Back + Main  

### C — Status / read-only
1. Title  
2. **One** `<table>` key/value with **expanded** values (never `{N keys}`)  
3. Optional 🔄 in body  
4. Keyboard: Back + Main  

## Global rules
- No `<tg-button>` inside `<td>`  
- `reply()` = edit-first; delete+sendRich fallback  
- EN/RU on every user-visible string  
- Catalog/Tools day-2 results must use A/B/C or dedicated formatter — **not** raw `formatSmart` for known actions  

## Screen matrix

| Screen | Template | Notes |
|--------|----------|-------|
| DNS lists | A | numbered on/off |
| Users list | A | # open card |
| User hub | B | emoji ops |
| Nodes list | A | |
| Node card | B | |
| Git list / repo | A / B | |
| Registry | B | |
| SSH hosts | A | |
| Backup list/schedule | A / B | |
| Pending / Locations | A / B | |
| NVR hub/cams/sites | B / A | |
| Catalog sections | A-like | body actions |
| **Status (main)** | **C** | metrics + services + nodes + VPN — **one screen** |
| Metrics / Health catalog | web-only | not separate TG menus |
| Doctor | C or checks table | |
| Guest / Edge / mTLS hubs | B | 0.9.41 |

## How to audit (for agents)
1. Open this matrix; any screen not listed = debt  
2. Grep `formatSmart(` in `internal/format` callers — known action IDs must not fall through  
3. Grep `inline_keyboard` in handlers: non-nav buttons under message = style break  
4. Before TG release: walk matrix in bot once  

## Changelog notes
- 0.9.45: metrics/status template C table with expanded cells; matrix documented  

- 0.9.46: Status absorbs host metrics; Metrics catalog TG removed; inventory [TG-SCREEN-INVENTORY.md](TG-SCREEN-INVENTORY.md)
