# Open items

## In progress

- TG style: matrix A/B/C documented (0.9.45); remaining screens audit ongoing
- Next: backbone WG spike; h2mux A/B; mTLS 8789 if offline alerts recur

- Day-2 UI parity + TG body-pattern closed through 0.9.41 (see REVIEW-0.9.41)
- Remaining: hardware e2e, owner later items below

## Ideas (no implementation yet)

### CDN-XHTTP plan B (RU CDN entry)

**Problem:** foreign VPS IP burned / mobile white-lists; direct Reality entry dead from some ISPs.

**Idea (2026):** client → **Russian CDN** (Yandex Cloud CDN / CDNVideo / …) → origin (EU VPS) with **VLESS + XHTTP** (`uplinkHTTPMethod: GET`, custom padding; Xray ≥ ~26.2.6, PR [#5414](https://github.com/XTLS/Xray-core/pull/5414)). Nginx on origin: secret path → core, else static decoy site + LE.

**Refs:** [ServerTechnologies/proxy-via-russian-cdn](https://github.com/ServerTechnologies/proxy-via-russian-cdn), video «Xray Vless XHTTP через российский CDN».

**vs netductor today:** stack is **sing-box + Reality** on secondary; this is a **separate optional entry**, not a replacement.

**If/when implement:**
1. Spike: does current **sing-box** support equivalent XHTTP GET/padding, or need Xray sidecar / dual-core?
2. Optional deploy path: domain + LE + nginx path + CDN CNAME (manual CDN account; no CF API required).
3. Client share links / profiles for Happ/SR that speak XHTTP.
4. Keep Reality as primary; CDN path as **failover profile** in users/TG.
5. Doc threat model: helps IP-block, weak vs domain block; latency ↑; CDN ToS/risk.

**Not doing now:** full automation, default install, Telegram-side CDN wizard.

## Owner / later

- Hardware e2e (OpenWrt, Tapo NVR, MikroTik+RPi)
- SMTP alerts
- Mobile day-2
- Rooms/photos Web model
- MikroTik API via Pi agent (discussion only)

## Done recently

- TG compact UI + catalog body actions / JSON (0.9.28–0.9.37)
- Web DNS + backup schedule forms; TUI Ops DNS/backup/git
- SSH hosts body actions; NVR lease pickers
- SNI health false-positive fixed
- Domain / LE / Mac-direct deploy

## Not doing

- Public VPS admin · Deploy from Telegram · Password day-2 tunnel


## Functional parity matrix (day-2)

| Capability | CLI | TUI | Web | TG |
|------------|-----|-----|-----|-----|
| Deploy primary/secondary/edge/site | ✅ | ✅ | ✅ Fleet | ❌ (by design) |
| Doctor | ✅ | ✅ | ✅ Control | ✅ catalog |
| Domain set/show | ✅ | ✅ Ops | ✅ | ✅ catalog |
| VPN users add/list/access | ✅ | ✅ | ✅ list/forms | ✅ compact |
| DNS block lists | ✅ | ✅ Ops | ✅ | ✅ |
| Backup schedule/list | ✅ | ✅ Ops | ✅ | ✅ |
| NVR cams/PTZ | ✅ | ✅ | Control | ✅ compact |
| Git/registry | ✅ | ✅ Ops | ✅ | ✅ body |
| Nodes card/ops | ✅ | ✅ | ✅ forms | ✅ |
| SSH hosts TOFU | ✅ | — | — | ✅ body |

**Parity pass (0.9.33–0.9.38):** Web Control + TUI Ops + TG pattern aligned for DNS, backup, git, catalog, SSH hosts.
