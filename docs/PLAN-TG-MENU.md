# PLAN: Telegram menu categories

**Status:** plan only (2026-10-08). Do not implement in the same change as channel alerts.

## Problem

Rich-text buttons and the reply keyboard both grew. After updates the hub stays in a topic, a second menu appears under it, and Tools repeats Status/Fleet rows. Operator cannot see one current menu at the bottom.

## Rules (already locked)

- Navigation buttons stay **under** the message (inline keyboard), not inside the card body, except deep links that must be rich-text.
- One hub message. After an alert, delete the old hub and send a new one **after** the alert, with a cooldown so a burst cannot recreate the hub in a loop.
- Card body is a table or short lines. Actions are buttons. No second copy of the same status in Tools.

## Categories

| Hub | Contains | Not here |
|--|--|--|
| Status | fleet digest, versions, channel, firewall | user add, logs export |
| Users | VPN users, policy, links, QR | node update |
| Fleet | nodes, routers, sites, addons | operator session |
| Tools | day-2 groups from opcatalog only (DNS, backup, logs, updates, git) | status table |
| Operator | session, alerts chat, audit, stack apply | end-user links |

## Checks before coding

1. Every `m:` callback maps to one screen. Unknown falls back to hub, not a second menu.
2. No row duplicated between hub and the screen it opens.
3. EN and RU labels from `T()`, not hardcoded in one screen only.
4. Topic bootstrap does not pin the hub inside "new topic".

## Done when

- After an alert, one hub sits at the bottom.
- Tools has no Status/Users duplicate.
- A checklist in this file is marked against the callback list.
