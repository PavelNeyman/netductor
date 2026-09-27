**EN** · [RU](ru/ARCHITECTURE-FREEZE.md)

# Architecture freeze (canon)

**Status:** locked. Feature work may proceed; topology/roles/paths below **must not** change without explicit owner decision.

## Canon (nails)

| Rule | Detail |
|------|--------|
| Roles | **primary** (abroad control) + **secondary** (RU VPN entry). Not full mirror of primary on RU |
| Users plane | VLESS Reality; clients → secondary → uplink primary |
| Service plane | Dual **SP/PS** WG-over-WSS (`10.87.10.0/30`, `10.87.11.0/30`); agent S→P on SP; recovery P→S on PS |
| API :8789 | mTLS only; WAN deny except service CIDRs + `api-allow.cidr`; temporary open **only** `api-public arm` (UI/CLI), **no** permanent env |
| Operator API | Loopback on node; **no** public node admin UI |
| Operator workstation | **Mac `netductor-op`** (TUI + local Web) over tunnel/VPN |
| SSH | **52222**, key-only after first bootstrap (**primary and secondary**; same Mac operator key) |
| Paths | FHS: `/usr/local/bin`, `/usr/local/share/netductor`, `/etc/netductor`, `/var/lib/netductor`. **No `/opt/netductor`** |
| Failover policy | Primary file of truth; secondary via agent `failover_policy` heartbeat |
| Backup | Config + data; binaries from release by component list |
| Redirect | **HTTPS :8443 only** (after LE). **No public :80**. Paths: `/r`, `/profiles/*`, `/healthz` only |

## Removed product surface (do not reintroduce)

- `NETDUCTOR_API_ALLOW_PUBLIC` permanent UFW open
- `NETDUCTOR_PLAIN_AGENT` plain :8788
- `NETDUCTOR_API_PUBLIC` non-local serve bind
- `NETDUCTOR_TRUST_PROXY` XFF trust
- Legacy admin UI on VPS (`LEGACY_ADMIN_UI`)
- `/opt/netductor` install prefix

## Before “features only” (plan)

See [ARCHITECTURE-PLAN.md](ARCHITECTURE-PLAN.md).

**Code (0.9.75):** removed knobs are not loadable via conf (`ndconfig` ignore list). Do not reintroduce enable paths.
