# PLAN: Telegram menu categories

**Status:** implemented in **v0.9.288**, nav fix **v0.9.289** (2026-10-08).

## Problem

Rich-text buttons and the reply keyboard both grew. After updates the hub stays in a topic, a second menu appears under it, and Tools repeats Status/Fleet rows. Operator cannot see one current menu at the bottom.

## Rules (locked)

- Navigation buttons stay **under** the message (inline keyboard), not inside the card body, except deep links that must be rich-text.
- One hub message. After an alert, delete the old hub and send a new one **after** the alert, with a cooldown so a burst cannot recreate the hub in a loop.
- Card body is a table or short lines. Actions are buttons. No second copy of the same status in Tools.

## Categories

| Hub | Contains | Not here |
|--|--|--|
| Status | fleet digest, versions, channel, firewall | user add, logs export |
| Users | VPN users, policy, links, QR | node update |
| Fleet | nodes, routers, sites, addons | operator session |
| Tools | day-2 groups from opcatalog only (Edge, Media, Data, Advanced + Updates/Logs) | status table, Users, Fleet, Alerts |
| Operator | session, alerts chat, audit, stack apply | end-user links |

## Checklist (callback map)

- [x] Hub top-level: `m:status` `m:users` `m:fleet` `m:tools` `m:operator` (+ lang/help)
- [x] Alerts only under Operator (`m:alerts-chat`), not on main hub row and not under Tools
- [x] Tools filters out home/users/fleet groups (`toolsDay2Groups`)
- [x] `m:cat:home` / `m:ops:overview` → Status + `statusKeyboard`, not Tools clone
- [x] Status includes firewall + channel one-liner; `m:channel` / `m:digest` / `m:versions` under Status
- [x] Operator includes Stack (`m:stack`) + Alerts + Session + Audit
- [x] Unknown callback → hub fallback (existing default)
- [x] EN/RU via `T()` for hub labels; Tools/Operator titles bilingual

## Done when

- [x] After an alert, one hub sits at the bottom (existing `repinHub` + batch)
- [x] Tools has no Status/Users duplicate
- [x] Checklist marked against the callback list


### 0.9.289 nav fix
- Status keyboard: Channel + Refresh only (not Versions/Digest→same Updates).
- `m:digest` → real fleet digest, parent Status.
- Updates only under Tools (`m:versions` / `m:updates`), back → Tools.


### 0.9.319 TG polish
- Help hub buttons use `T()` (no mixed EN/RU labels).
- Status: channel/refresh/incident via i18n keys.
- Users: presets/canary/devices via `T()`; canary hub back label bilingual.
- No extra top-level hubs; Tools remains Day2 only.

## Structure pass (2026-10-12)

Inventory of `m:` callbacks and hub map. Findings and fixes:

| Issue | Fix |
|--|--|
| DR screen button labeled **Fleet** but callback `m:digest` (misleading duplicate destination) | Label → `T("digest_btn")` |
| Digest/DR action buttons hardcoded EN (`Updates`, `DR`, `Stack`, `Menu`) | Switched to `T("updates_btn")`, `T("dr_btn")`, `T("stack_btn")`, `T("main_menu")`, `T("refresh_btn")` |
| `m:updates` and `m:versions` both open Updates | Intentional alias (Tools uses `m:versions`; digest shortcut uses `m:updates`). Same screen, different entry points — kept. |
| `topicsKeyboard` still exists | Redirects to alerts chat (legacy callbacks). Kept for old messages. |
| Tools | Still day-2 only via `Day2Groups()` (skips home/users/fleet). No Status clone. |
| Hubs | Status / Users / Fleet / Tools / Operator unchanged. Alerts under Operator. |

Checklist:
- [x] Inventory all `m:` callbacks and handlers
- [x] Map button → screen; flag duplicates
- [x] Remove/fix misleading path (DR “Fleet” → digest)
- [x] Back/menu present on hub keyboards (`m:menu`)
- [x] User-visible labels on fixed screens via `T()`
