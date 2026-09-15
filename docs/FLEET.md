# Fleet: primary & secondary

## Naming

| Operator term | Typical host | Internal notes |
|---------------|--------------|----------------|
| **primary** | Abroad VPS | Control plane: users, policies, TG bot active, admin, backup authority. Hostname `nd-primary` |
| **secondary** | RU VPS | VPN entry under WL + warm services (Lampac). Hostname `nd-secondary`. VPN agent code path still uses `role=relay` |

Do **not** load-balance VPN. Secondary is the default client entry; primary is the source of truth.

## Deploy secondary from primary

```bash
# on primary (after core install/recover)
netductor fleet provision-secondary \
  --host 92.x.x.x --password '…' [--sni api.vk.me]

# equivalent low-level (VPN only):
# netductor relay provision --host … --password …
```

`provision-secondary` runs:

1. VPN join (`relay provision` + post hooks)
2. Fleet roles (`secondary`, desired hostname `nd-secondary`)
3. Data sync (Lampac volume, policy)
4. Lampac on secondary (unless `--no-lampac`)
5. Bot standby units on secondary (unless `--no-bot-standby`) — SOCKS via primary
6. Hourly `fleet sync` timer on primary

## Commands

```bash
netductor fleet status|bootstrap
netductor fleet provision-secondary --host IP --password PASS
netductor fleet sync | sync-timer | apply-lampac
netductor fleet bot-failover check|promote|demote|timer
```
