# Fleet: primary & secondary

See also: [AGENT_HANDOFF.md](AGENT_HANDOFF.md) · [BACKUP.md](BACKUP.md) · [DEPLOY.md](DEPLOY.md)

## Naming

| Operator term | Typical host | Internal notes |
|---------------|--------------|----------------|
| **primary** | Abroad VPS | Control plane: users, policies, TG bot active, admin, backup authority. Hostname `nd-primary` |
| **secondary** | RU VPS | VPN entry under WL + warm services (Lampac). Hostname `nd-secondary`. VPN agent code path still uses `role=relay` |

Do **not** load-balance VPN. Secondary is the default client entry; primary is the source of truth.

## Deploy secondary from primary

```bash
netductor fleet provision-secondary \
  --host 92.x.x.x --password '…' [--sni api.vk.me] \
  [--no-lampac] [--no-bot-standby]
```

Steps: VPN join → fleet roles → data sync → Lampac → bot standby (SOCKS→primary) → hourly sync timer.

VPN-only: `netductor relay provision --host … --password …`

## TG failover

- Active bot on **primary**
- Standby on **secondary** only if primary host is up but bot unit is down
- Exit via `ssh -D` SOCKS to primary (`/api/bot-status` on `:8788`)
- If primary host is dead, standby is **not** used (SOCKS cannot exit via dead core)

## Commands

```bash
netductor fleet status|bootstrap
netductor fleet provision-secondary --host IP --password PASS
netductor fleet sync | sync-timer | apply-lampac
netductor fleet bot-standby-install [user@primary]
netductor fleet bot-failover check|promote|demote|timer
```


## Implementation notes

- Do **not** wipe relay `devices.json` tokens in post-provision before the first successful agent heartbeat (token is issued on primary and stored on secondary).
- Agent auth: `Authorization: Bearer <relay_agent_token>` → primary `:8788`.
