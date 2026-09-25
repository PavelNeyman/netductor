# AGENTS.md — Netductor

> Document Version: **2.6**  
> Status: **Approved**  
> GitHub: **https://github.com/PavelNeyman/netductor** (renamed from FreshVPS)

**Single source of truth for project rules and architecture.**  
**Conversation history must never replace this document.**

**Release baseline: v0.9.0** (operator/node binary split) · Handoff: [docs/AGENT_HANDOFF.md](docs/AGENT_HANDOFF.md)

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
8. **Go only** for control plane (install/serve/vpn/doctor/agent/tg).
**Binaries:** `netductor-op` (workstation) vs `netductor` node (`netductor-linux-*` on VPS). Shell/Python under `legacy/` is reference-only, not runtime.
9. One logical Git commit per task.

---



---

# Documentation & checklist discipline (**mandatory for AI**)

This rule is **strict and non-negotiable**:

1. Work follows [docs/OPERATOR-PLAN.md](docs/OPERATOR-PLAN.md) (and ROADMAP when relevant).
2. When any checklist item is **finished**, the **same** change (commit / PR / task outcome) **must**:
   - mark that item `[x]` in the plan file;
   - update affected docs (ARCHITECTURE-OPERATOR, DOMAIN, DEPLOY-MAC, HANDOFF, CHANGELOG as applicable);
   - keep EN+RU in sync when user-facing behaviour is documented in both.
3. **Forbidden:** closing a task as done while leaving plan boxes unchecked or handoff/architecture stale.
4. **Forbidden:** implementing deploy behaviour only in TUI or only in CLI — extend operator/deploy Spec + one use-case, then thin UI.
5. New chats: read [docs/ARCHITECTURE-OPERATOR.md](docs/ARCHITECTURE-OPERATOR.md) before changing deploy/TUI.

Operator vs node and phases: [docs/ARCHITECTURE-OPERATOR.md](docs/ARCHITECTURE-OPERATOR.md).

---

# Editing AGENTS.md

**Do not modify without explicit owner consent in the current task.**

---

# 1. Product

**Netductor** — personal network control plane: VPN, DNS, edge OpenWrt agents, CLI / Admin / Telegram.

Planes: **host** · **vpn** (sing-box) · **dns** (Blocky) · **core/API** · **edge** · **operator** · **fleet** (primary/secondary) · **extras** (optional).

**Fleet:** primary = abroad control plane; secondary = RU VPN entry + warm services. Details: [docs/FLEET.md](docs/FLEET.md), [docs/AGENT_HANDOFF.md](docs/AGENT_HANDOFF.md). Secondary was formerly called `relay` (legacy paths/API aliases remain).

**SSH:** password only for first login; install/provision → key-only (`internal/install/ssh_harden.go`).

**TG UI:** navigation under the message; screen actions in HTML body — [docs/TG-UI.md](docs/TG-UI.md). Access import buttons use **redirect-serve** + `NETDUCTOR_REDIRECT_BASE` (HTTPS **:8443** after LE; not custom schemes in Telegram url-buttons).

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
| Version | **0.8.30** |
| G0–G1 | done |
| G2 | release `v0.8.30` published |
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

### 0.8.0 release
See CHANGELOG and docs/REVIEW-2026-09-20.md. Update policy: docs/UPGRADE.md.

Current backlog: [docs/OPEN_ITEMS.md](docs/OPEN_ITEMS.md). Handoff: [docs/AGENT_HANDOFF.md](docs/AGENT_HANDOFF.md).
