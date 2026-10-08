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
