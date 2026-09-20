# Fleet: primary + secondary

## Model (locked)

| Node | Role |
|------|------|
| **primary** (abroad) | Control plane: users, API, TG bot, Blocky, edge enroll, backups, optional **Lampac** |
| **secondary** (RU) | **VPN entry only** (VLESS/Reality + thin agent). Hostname `nd-secondary`. |

VPN is **not** load-balanced. Clients prefer secondary under carrier whitelist.

Internal agent code may still say `role=relay` — same plane, operator name is **secondary**.

See [PLAN-SECONDARY-VPN-ONLY.md](PLAN-SECONDARY-VPN-ONLY.md).

## What is NOT on secondary

- Lampac / Docker service mirror
- Telegram bot standby / failover
- Hourly `fleet sync` of app data
- Edge enroll API (agents → primary)

## What IS synced to secondary

- **VPN user UUIDs** via `ApplyConfig` → `config_ver` bump → agent pulls `ExportRelayBundle` into `relay-in`
- Force: `netductor secondary sync`

## Commands

```bash
netductor fleet status
netductor fleet bootstrap
netductor fleet provision-secondary --host IP --password '…' [--sni api.vk.me]
netductor fleet disable-legacy   # stop old sync/failover timers on this host
netductor install lampac         # primary only
netductor secondary sync             # push VPN users to secondary now
```

## Policy file

`/var/lib/netductor/fleet/policy.json` — `sync_enabled` defaults **false**.


## TUI

Tools: **VPN → secondary** (`secondary sync`), Lampac on primary, wizard secondary = VPN entry only.


## Remote ops from laptop

```bash
# flags
netductor tui --remote PRIMARY_IP --remote-user root --remote-key ~/.ssh/id_ed25519

# or Settings tab in TUI (saved to ~/.config/netductor/tui.yaml)
# or env: NETDUCTOR_REMOTE / NETDUCTOR_REMOTE_USER
```
