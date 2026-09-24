# Operator core — implementation checklist

**Source of truth for progress.** AI agents: see mandatory update rule in [AGENTS.md](../AGENTS.md).

Legend: `[ ]` todo · `[x]` done · `[~]` in progress

## Phase 0 — Documentation

- [x] [ARCHITECTURE-OPERATOR.md](ARCHITECTURE-OPERATOR.md) — target model + phases
- [x] Compress [AGENT_HANDOFF.md](AGENT_HANDOFF.md) — drop version archaeology
- [x] AGENTS.md — operator rules + **mandatory doc update on every completed item**
- [x] This checklist created

## Phase 1 — Use-case layer

- [ ] Package `internal/operator` (name flexible) with Spec types shared by CLI/TUI
- [ ] `DeployPrimary` / `DeploySecondary` / `CollectCredentials` invoked **only** via operator package
- [ ] CLI `deploy primary|secondary` thin wrapper over Spec
- [ ] TUI Fleet + Primary + Secondary thin wrapper over Spec (no private opts divergence)
- [ ] Single place for domain/LE/add-ons/TG field → Spec mapping helpers
- [ ] Docs: update ARCHITECTURE-OPERATOR + DOMAIN + DEPLOY-MAC if CLI flags change
- [ ] Mark items `[x]` here when done

## Phase 2 — FleetDeploy

- [ ] `FleetDeploy(FleetSpec)` one function: ordered primary → secondary → credentials
- [ ] Structured step events (or stable step IDs) for progress UI
- [ ] TUI Fleet calls `FleetDeploy` only
- [ ] CLI `deploy fleet` (or equivalent) same entry
- [ ] Docs + this checklist updated

## Phase 3 — Localhost API + WebUI

- [ ] `netductor operator serve` bind **127.0.0.1 only**
- [ ] JSON endpoints for fleet/primary/secondary + event stream
- [ ] `go:embed` minimal WebUI (Fleet form + log + credentials path)
- [ ] Security note in docs (loopback only)
- [ ] Docs + this checklist updated

## Phase 4 — Optional

- [ ] Binary split `netductor` (node) vs `netductor-op` (operator)
- [ ] Mobile day-2 client (not bootstrap)

## Done recently (context, not phase work)

- [x] v0.8.94 — legacy huh deploy removed; framed Fleet wizard; primary TUI fields ≈ CLI
- [x] Domain/LE on primary deploy path; credentials on Mac after deploy
