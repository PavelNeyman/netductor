# Phase 3 — Control standby (notes, not implemented)

**Status:** design only (2026-11-17). No second control writer in production path yet.

## SoT datasets to replicate (minimum)

| Path / area | Why |
|-------------|-----|
| `/etc/netductor/vpn-users.json` (+ policy) | Users / access |
| `/etc/netductor/secrets/` (CA, Reality, backup_key, TG, …) | Identity |
| `/var/lib/netductor/secondary/devices.json` | Egress registry |
| `/etc/netductor/svc-paths/` | Peer material / active_id |
| Domain / redirect conf | Client entry URLs |
| Stack pin / VERSION | Upgrade coherence |

Prefer **encrypted backup pull** already used for offsite rather than inventing a second wire format first.

## Promote control (intended)

1. Standby has fresh backup + secrets.
2. Explicit `control promote` (to design) with fencing: old active API refuse writes.
3. Agents: ordered list of control endpoints (mTLS URL).
4. DNS/API host cutover documented like VPN host.

## Non-goal until Phase 2 ops are routine

Multi-master SoT (Phase 5).
