# TG SPA menu parity (classic hubs)

Flag: `NETDUCTOR_TG_WIDGETS=1`. Entry: `/start`, `/menu`, `m:menu`.

## Top level (= mainKeyboard)

| Classic | SPA screen |
|---------|------------|
| Status | `status` |
| Users | `users` → `ulist` / `devices` |
| Fleet | `fleet` |
| Tools | `tools` |
| Operator | `operator` |
| Lang | `L:ru` / `L:en` |
| Help | `help` |

## Status (= statusKeyboard)

channel, netmap, actions, sessions, digest, incident, reboot(hint→node card)

## Fleet (= fleetHubHTML)

nodes, routers*, sites*, secondary, addons  
\*routers/sites also under Tools→Edge

## Tools (= Day2 + extras)

- **edge** → routers, sites, pending  
- **media** → lampac, canary, nvr  
- **data** → backup, logs, dns  
- **adv** → probes, stack, doctor, certs, fw, cleanup  
- versions, logs, ssh, git  

## Operator (= operatorHubHTML)

Nav: session, alerts, sessions, audit, svcpaths, stack, admin  
Acts: API WAN arm/disarm 15m, Users→SP on/off, VPN refresh links  

## Cards

- Users list `#` → `ucard` (enable/disable/link/sub/revoke)  
- Nodes list `#` → `ncard` (upgrade/reboot/journal)  

## Intentionally still classic (input / multi-step / media)

rename user, quota N, QR photo, git project flows, edge luci/guest wizards,  
backup restore confirm paths, stack heal/rollback confirm, reboot challenge codes,  
alerts-chat pick, text prompts (add user, ssh forget, …).
