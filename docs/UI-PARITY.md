# UI parity (locked model)

**Verified through v0.9.42** (review 0.9.41) — see [REVIEW-0.9.39.md](REVIEW-0.9.39.md).

## Rule

**One backend** (`internal/operator` + `internal/deploy` + node API).  
**All UIs are thin:** Web, TUI, CLI — same use-cases, same outcomes.

| UI | Role |
|----|------|
| **Web** (`operator serve`) | Full Installer + Control |
| **TUI** | Full Wizard + Tools (same Deploy* / CLI) |
| **CLI** | Same packages / `netductor` on node / op subcommands |
| **TG** | Day-2 only on node — **no fleet deploy** (ops channel, not installer) |

## Deploy parity (Mac)

| Use-case | Web | TUI | CLI/op |
|----------|-----|-----|--------|
| Fleet / Primary / Secondary | ✅ | ✅ | ✅ |
| Telegram on primary (token+admin) | ✅ Fleet/Primary fields | ✅ Add-ons / fields | ✅ `--tg-token` |
| Credentials collect | ✅ | ✅ | ✅ |
| OpenWrt edge (`DeployEdge`) | ✅ `/v1/edge` | ✅ wizard | ✅ |
| MikroTik site (`DeploySite`) | ✅ `/v1/site` | ✅ wizard | ✅ |
| MikroTik site | gap → close via same backend | ✅ partial | CLI |
| NVR day-2 | Control API | Tools + wizard | `netductor nvr` |

Any “TUI-only” hardware deploy is a **parity debt**, not product intent.

## Day-2

Web Control + Advanced, TUI Tools, TG Tools, node CLI — all talk to the **same node API / CLI**.

## Action catalog (0.9.18)

Go package `internal/opcatalog` — single registry of day-2 session actions.
- Operator: `GET /v1/catalog`
- Web Control loads catalog for button labels/paths
- TG: same package available for progressive migration of menus

## NVR Web Installer

Installer tab **NVR** mirrors TUI wizard actions (status/list/leases/add/record) via node session API.
- TG Tools hub: opcatalog sections (`m:ops:` / `m:op:`)
