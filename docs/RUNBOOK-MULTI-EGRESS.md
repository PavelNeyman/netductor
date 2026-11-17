# Runbook: second egress (multi-egress)

**Goal:** two (or more) VPN entry nodes; clients keep a working path if one dies.  
**Not automatic:** peer **promote**, DNS A for `vpn_hostname`, and retiring the old node stay **operator steps**.

## Prerequisites

- Control (legacy primary) healthy: API, mTLS, `/sub` / redirect.
- New VPS pre-checked (`vps-check` / wizard precheck).
- Same Reality SNI policy as the fleet (e.g. `api.vk.me`).
- Operator key on Mac; deploy via **op** (Web / TUI / CLI), not via SSH from control.

## Add B

1. Deploy egress B with **Promote=false** (default). Region e.g. `ru-msk`.
2. Confirm registry: B `prepared`, heartbeat online, `singbox_ok`.
3. `/sub` should list B (prepared/ready) **and** A (active). Prefer B for your client:
   ```bash
   netductor secondary prefer <B_public_ip>
   ```
4. Import/refresh subscription; test **only** the B profile (Reality to B IP).
5. Service plane: `svc-paths peer ready` / validate as documented for your slot.
6. When traffic on B is OK:
   ```bash
   netductor svc-paths peer promote <peer-id-B>
   ```
7. Point DNS A of `VPN_HOST` / `vpn_hostname` at B **after** promote if clients use the name.
8. Keep A online until sub + real users are stable; then retire A (remove peer only after promote away from it).

## Health vs promote

| Signal | Effect |
|--------|--------|
| Heartbeat stale | Not in `/sub` |
| `singbox_ok=false` | Not in `/sub` |
| `uplink_ok=false` | Still in `/sub` (sort demotion) |
| `secondary prefer` | Order only |
| `peer promote` | Service-plane **active** peer |

## Rollback

- Prefer A again: `secondary prefer <A_ip>`; if B was promoted, `svc-paths peer rollback` / promote A per CLI help.
- DNS: restore previous A record if cut over.

## Limits

- Control outage does not instantly kill existing client sessions on egress (best-effort).
- Auto-promote is **out of scope** until explicit R5-class gates exist (see fleet plan Phase 2).
