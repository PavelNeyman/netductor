# AGENTS.md — Netductor

> Document Version: **2.4**  
> Status: **Approved**  
> GitHub: **https://github.com/PavelNeyman/netductor** (renamed from FreshVPS)

**Single source of truth for project rules and architecture.**  
**Conversation history must never replace this document.**

Progress: [docs/ROADMAP.md](docs/ROADMAP.md) · [docs/ru/ROADMAP.md](docs/ru/ROADMAP.md) · [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md)

---

# Product category

**Production (self-hosted, single-operator).** Not multi-tenant SaaS.

---

# Quick Start for AI Agents

1. Read this AGENTS.md entirely.
2. Read [docs/AGENT_HANDOFF.md](docs/AGENT_HANDOFF.md) + ROADMAP + ARCHITECTURE + FLEET.
3. Inspect **github.com/PavelNeyman/netductor** (bins in `/usr/local/bin`; `/opt/netductor` = data volumes only).
4. Respect Forbidden / Frozen Architecture.
5. If ambiguous → **STOP** and ask the owner.
6. Implement only the requested task.
7. Docs EN+RU when user-facing behaviour changes.
8. **Go only** for control plane (install/serve/vpn/doctor/agent/tg). Shell/Python under `legacy/` is reference-only, not runtime.
9. One logical Git commit per task.

---

# Editing AGENTS.md

**Do not modify without explicit owner consent in the current task.**

---

# 1. Product

**Netductor** — personal network control plane: VPN, DNS, edge OpenWrt agents, CLI / Admin / Telegram.

Planes: **host** · **vpn** (sing-box) · **dns** (Blocky) · **core/API** · **edge** · **operator** · **fleet** (primary/secondary) · **extras** (optional).

**Fleet:** primary = abroad control plane; secondary = RU VPN entry + warm services. Details: [docs/FLEET.md](docs/FLEET.md), [docs/AGENT_HANDOFF.md](docs/AGENT_HANDOFF.md). Secondary was formerly called `relay` (legacy paths/API aliases remain).

**SSH:** password only for first login; install/provision → key-only (`internal/install/ssh_harden.go`).

**TG UI:** navigation under the message; screen actions in HTML body — [docs/TG-UI.md](docs/TG-UI.md). Access import buttons use **:80 redirect-serve** (`NETDUCTOR_REDIRECT_BASE`); do not put custom schemes in Telegram url-buttons.

---

# 2. Philosophy

Simplicity, idempotent installs, no secrets in repo, releases ship binaries, evolution over big-bang.

---

# 3. Architecture (frozen direction)

| Item | Choice |
|------|--------|
| Repo | `PavelNeyman/netductor` |
| CLI | `netductor` |
| Agent | `netductor-agent` |
| VPN | sing-box only |
| DNS | Blocky |
| Edge | own outbound agent (OpenSOHO optional) |
| Monitoring | built-in metrics/probes |
| Docs | EN + RU |

**Domain:** `NETDUCTOR_PUBLIC_HOSTNAME` (mTLS SAN), `NETDUCTOR_VPN_HOST` (client links).

**Paths (G7):**
- Config/secrets: `/etc/netductor`
- State: `/var/lib/netductor` (devices under `secondary/`; `relay/` read for migration)
- Binaries: `/usr/local/bin/netductor`, `netductor-tg`
- Data only under `/opt/netductor`: lampac volume, admin static (`runtime/api/admin`)


**Forbidden without approval:** replace sing-box/Blocky; default-on Kuma/Beszel/Lampac; secrets in git; multi-tenant SaaS; delete bash without Go replacement.

---

# 4–7. Tech / layout / git / DoD

Go orchestrator; bash bridge; default branch **main**; no secrets; doctor/smoke meaningful; EN+RU docs.

Target layout: `cmd/netductor`, `cmd/netductor-agent`, `internal/`.

---

# 8. Project State

| Area | Status |
|------|--------|
| Repo | **PavelNeyman/netductor** |
| Version | **0.7.2-dev** |
| G0–G1 | done |
| G2 | release `v0.7.0-dev` published |
| G3 | `netductor serve` health scaffold; port Python API |
| Next | OpenWrt/MikroTik e2e; optional HTTPS |

---

Final: production quality, chat never overrides AGENTS.

## Node naming (fleet)

Format: `nd-<role>-<marker>` (e.g. `nd-core-nl01`).
Roles: `core` | `secondary` | `edge` | `lab`. Marker: region+number or IP suffix.
See [NODES.md](NODES.md). Registry is bidirectional (device heartbeat ↔ operator desired hostname).
UI hints: Admin → Nodes, Telegram → Nodes, TUI → Set hostname.


## Site / MikroTik
- TUI wizard: workstation|operator → Site setup wizard
- Credentials one-shot, never stored
- VPN only on RPi OpenWrt, not ROS

## NVR

See `docs/PLAN-NVR-TAPO.md`. CLI: `netductor nvr`. API under `/api/nvr/*`. Retention defaults: 7d / 40GB / 5GB free.

### 0.7.36-dev — relay name removed
- State: only `secondary/` (no `relay/` fallback).
- CLI: `netductor secondary` only (no `relay` alias).
- API: `/api/secondary/*` only.
- Node role/id prefix: `secondary` / `secondary-…`.
