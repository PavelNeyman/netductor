# Fleet roles (core + relay)

## Decision

| Concern | Node | Why |
|--------|------|-----|
| **Control-plane primary** | **Abroad core** | Users, policies, TG operator bot, admin, backup authority |
| **VPN entry** | **RU relay** | Whitelist / mobile |
| **Lampac (default)** | **RU** | Latency; data synced hourly |
| **TG bot** | **Abroad active**; RU **standby** via SOCKS→core | TG often blocked in RU; Bot API needs HTTPS exit abroad |

## TG failover (no MTProxy)

Bot API is HTTPS to `api.telegram.org`. MTProxy is for client apps, not required here.

On **secondary (RU)**:
1. `ssh -D 127.0.0.1:1089` to primary → SOCKS exits with **core IP**
2. Standby unit: `ALL_PROXY=socks5://127.0.0.1:1089 netductor-tg`
3. Timer every 2 min: if primary bot not `active` → start standby; if primary healthy → stop standby

```bash
# on primary
netductor fleet sync-timer
netductor fleet apply-lampac

# on secondary (once)
netductor fleet bot-standby-install root@CORE_IP
netductor fleet bot-failover timer
```

## Commands

```bash
netductor fleet bootstrap|status
netductor fleet sync | sync-timer | apply-lampac
netductor fleet bot-standby-install [user@primary]
netductor fleet bot-failover check|promote|demote|timer
```
