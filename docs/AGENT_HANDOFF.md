# Agent handoff (netductor)

**Repo:** https://github.com/PavelNeyman/netductor  
**Model:** primary (abroad) + secondary (RU entry). Operator UI: **Mac** `netductor-op` (TUI + local Web). Node: `netductor` on VPS.

## Locked architecture

| Plane | Path | Role |
|-------|------|------|
| Users | VLESS Reality (secondary→primary uplink) | client traffic |
| Service SP | WG-over-WSS `nd-svc-sp` 10.87.10.0/30 | agent S→P, API via tunnel |
| Service PS | WG-over-WSS `nd-svc-ps` 10.87.11.0/30 | primary→secondary (recovery pull) |
| API :8789 | mTLS; WAN **deny** except service CIDRs + `api-allow.cidr` | TG `api-public` toggle opens 15m |
| SSH | port **52222**, key-only after harden | break-glass |

**Failover:** `svc-paths failover` policy on primary; synced to secondary via agent heartbeat `failover_policy`. Users→SP: sing-box uplink server `2.27.x` ↔ `10.87.10.1` when public :443 fails (policy flag).

**Paths (FHS):** `/usr/local/bin/*`, `/usr/local/share/netductor/*`, `/etc/netductor`, `/var/lib/netductor`. **No `/opt/netductor`.**

## Operator (Mac)

- Binary: `netductor-op` (brew tap `pavelneyman/netductor`)
- **TUI** + **WebUI** (localhost): same `internal/deploy` + remote API over SSH tunnel or VPN
- Deploy wizards: primary → secondary → domain/LE → addons; credentials saved on Mac
- Parity rule: thin clients over one backend; new API → expose in opcatalog + TUI + Web

## Telegram

- Operator-only bot on primary; EN/RU
- Templates A/B/C: [TG-UI-PATTERN.md](TG-UI-PATTERN.md)
- Operator hub: **table** state (API public, Users→SP) + **toggle** buttons (single control per action)
- Policy change propagates to secondary on next agent heartbeat (~15–30s)

## CLI highlights

```
netductor api-public arm|disarm|status
netductor svc-paths status|apply|failover …
netductor cleanup-legacy --apply
netductor recovery arm|disarm  # secondary
```

## Security notes

- Do not enable `NETDUCTOR_API_ALLOW_PUBLIC` permanently; use TG toggle
- Do not enable `PLAIN_AGENT` / `API_PUBLIC` env footguns in prod
- Recovery :8790 only when armed
- Trust `X-Forwarded-For` only if `NETDUCTOR_TRUST_PROXY=1`

## Next engineering (open)

1. Optional TG alert when `user_path=degraded`
2. Full opcatalog parity audit (NVR/guest/git) on Web vs TUI
3. Cert expiry WARN in doctor (if not already)
4. Hardware e2e OpenWrt/Tapo (needs iron)

## Docs map

| Doc | Topic |
|-----|--------|
| PATHS.md | FHS layout |
| BACKBONE-WG.md | SP/PS, failover |
| PORTS.md | listening ports + api-public |
| TG-UI-PATTERN.md | bot screen rules |
| DEPLOY.md / INSTALL.md | install |
| OPEN_ITEMS.md | backlog |

**Rule:** every completed item → update this handoff + mark OPEN_ITEMS; push release bins when shipping to VPS.
