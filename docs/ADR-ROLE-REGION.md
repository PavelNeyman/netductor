# ADR: Role + region naming (CF-14)

**Status:** accepted 2026-11-06; **amended 2026-11-17** (geography orthogonal; region shape)  
**Context:** operators and clients need stable names for management vs VPN-attachment nodes and a geography label in subscription profiles. The old in/out framing was rejected.  
**Evolution plan:** [PLAN-FLEET-MESH-EVOLUTION.md](PLAN-FLEET-MESH-EVOLUTION.md)

## Decision

| Axis | Canonical values | Legacy aliases |
|------|------------------|----------------|
| **Role** | `control`, `egress`, `build`, `edge` | `primary`≡`control`, `secondary`≡`egress` |
| **Region** | placement label, not a role | optional; set at deploy |
| **Instance** | peer id + slot | blue-green within a region |

### Role is functional, not geographic

- **`control`** — source of truth and management plane (API, policy, users, orchestration, backups coordination). **Not** defined as “abroad” or “EU”.
- **`egress`** — user VPN attachment and/or traffic path (ingress for clients, egress path per policy). **Not** defined as “RU-only”.
- Topology *control in one country + egress in another* is a **common deployment template**, not part of the role definition.
- Future model may emphasize **capabilities** (`vpn_ingress`, `build`, …) on a `member` node; role ids above remain for compatibility (see fleet plan).

### Region shape (amended)

Preferred form:

```text
<iso3166-1-alpha2>-<city-slug>    # ru-msk, fi-hel, us-nyc, de-fra
```

Short labels (`eu`, `ru`) remain acceptable during transition but new deploys should prefer `cc-city`.

### Hostnames (convention)

```text
nd-<role>-<region>    # nd-control-fi-hel, nd-egress-ru-msk
```

## Compatibility

- On-disk `/etc/netductor/role` continues to accept **and** (for a long transition) **write** `primary`|`secondary` so older binaries keep working (`role.LegacyWrite`).
- Code dual-reads via `role.Normalize` / `role.AcceptsRoleFile`.
- CLI/API keep `primary`/`secondary` subcommands and flags.

## Non-goals

- Renaming every internal Go package or WireGuard interface in one shot.
- Forcing region on every peer immediately.
- Multi-master control plane as a prerequisite of this ADR (see fleet plan Phase 5).

## Follow-ups

- `region` on peer registry and subscription profile display names (**Phase 1** of fleet plan).
- Web/TG/doctor labels using canonical terms with legacy still shown where helpful.
- Capabilities array / flags for non-VPN workloads on member nodes.
- ARCHITECTURE / FREEZE wording: example topologies vs role definitions.
