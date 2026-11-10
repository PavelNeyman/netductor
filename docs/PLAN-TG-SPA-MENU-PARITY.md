# TG SPA menu parity (classic hubs)

Flag: `NETDUCTOR_TG_WIDGETS=1`. Entry: `/start`, `/menu`, `m:menu`.

## Top level (= mainKeyboard)

| Classic | SPA screen |
|---------|------------|
| Status | `status` |
| Users | `users` → `ulist` / `devices` / **Add** |
| Fleet | `fleet` |
| Tools | `tools` |
| Operator | `operator` |
| Lang | `L:ru` / `L:en` |
| Help | `help` |

## Status

channel, netmap, actions, sessions, digest, incident, **reboot** (pick → challenge codes)

## Fleet

nodes, routers, sites, secondary, addons

## Tools (Day2)

edge / media / data / adv + versions, logs, ssh, git

## Operator

session, alerts (**test/clear**), sessions, audit, svcpaths, stack (**heal / rollback confirm / watchdog**), admin  
Acts: API WAN, Users→SP, VPN refresh links

## User card

enable, disable, link, sub, **quota 50/200/∞**, **rename** (hybrid: next text message), revoke+confirm

## Node card

upgrade, **reboot challenge**, journal

## Backup

list `#` → restore confirm

## Hybrid text input (classic message state)

- Rename: SPA sets `wait_rename_new:` → user sends name in chat  
- Add user: SPA sets `wait_vpn_add_name` → user sends name  

## Still limited / deep wizards

- QR as photo (link text in SPA; photo stays classic)  
- Git project multi-step ship/migrate  
- Edge luci/guest multi-step wizards  
- Alerts-chat pick from list of known chats (test/clear done)  
- Custom quota N (preset 50/200/∞ in SPA; custom still classic text)
