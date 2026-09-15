# Fleet roles (core + relay)

## Decision

| Concern | Node | Why |
|--------|------|-----|
| **Control-plane primary** (users, policies, TG operator bot, admin, backup authority) | **Abroad core** | Secrets less exposed; Telegram API more reliable; already the source of truth |
| **VPN entry (data plane)** | **RU relay** | Whitelist / mobile reachability |
| **Lampac (default placement)** | **RU** (prefer) | Lower latency for users in RU; warm replica synced from/to primary |
| **TG bot** | **Abroad** (active), RU = standby later | One bot token; active/passive only |

VPN tunnels are **not** load-balanced. Service placement and data sync are separate.

## Commands

```bash
netductor fleet bootstrap          # primary=core, secondary=relay, lampac→RU, bot→core
netductor fleet status
netductor fleet set-primary <id>
netductor fleet set-secondary <id>
netductor fleet set-service lampac <id>
netductor fleet set-service bot <id>
netductor fleet sync               # rsync/scp lampac data + fleet policy to backup peer
```

## Sync

`fleet sync` copies:

- `/opt/netductor/lampac` (config/progress, not the image)
- `components.json`, `fleet/` policy

Peer host is taken from `backup peer-set` if not passed explicitly.
