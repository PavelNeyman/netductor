# ADR: Role + region naming (CF-14)

**Status:** accepted 2026-11-06  
**Context:** operators and clients need stable names for control-plane vs VPN-entry nodes and a geography label in subscription profiles. The old in/out framing was rejected.

## Decision

| Axis | Canonical values | Legacy aliases |
|------|------------------|----------------|
| **Role** | `control`, `egress`, `build`, `edge` | `primary`≡`control`, `secondary`≡`egress` |
| **Region** | user-facing geo (`eu`, `ru`, …) | optional; set at deploy |
| **Instance** | peer id + slot | blue-green within a region |

## Compatibility

- On-disk `/etc/netductor/role` continues to accept **and** (for a long transition) **write** `primary`|`secondary` so older binaries keep working (`role.LegacyWrite`).
- Code dual-reads via `role.Normalize` / `role.AcceptsRoleFile`.
- CLI/API keep `primary`/`secondary` subcommands and flags.

## Non-goals

- Renaming every internal Go package or WireGuard interface in one shot.
- Forcing region on every peer immediately.

## Follow-ups

- Optional `region` on peer registry and subscription profile display names.
- Web/TG/doctor labels using canonical terms with legacy still shown where helpful.
