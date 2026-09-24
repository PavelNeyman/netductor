# Operator core — implementation checklist

**Source of truth for progress.** AI agents: see mandatory update rule in [AGENTS.md](../AGENTS.md).

Legend: `[ ]` todo · `[x]` done · `[~]` in progress

## Phase 0 — Documentation

- [x] [ARCHITECTURE-OPERATOR.md](ARCHITECTURE-OPERATOR.md) — target model + phases
- [x] Compress [AGENT_HANDOFF.md](AGENT_HANDOFF.md) — drop version archaeology
- [x] AGENTS.md — operator rules + **mandatory doc update on every completed item**
- [x] This checklist created

## Phase 1 — Use-case layer

- [x] Package `internal/operator` with Spec types shared by CLI/TUI
- [x] `DeployPrimary` / `DeploySecondary` / `CollectCredentials` via operator package
- [x] CLI `deploy primary|secondary` thin wrapper over Spec
- [x] TUI Fleet + Primary + Secondary thin wrapper over Spec
- [x] `PrimaryFromFields` / `FleetFromFields` / `ApplyDomainFlags` helpers
- [x] Docs updated
- [x] Mark items `[x]` here when done

## Phase 2 — FleetDeploy

- [x] `FleetDeploy(FleetSpec)` one function: ordered primary → secondary
- [x] TUI Fleet calls `FleetDeploy` only
- [x] CLI `deploy fleet` same entry
- [ ] Structured step events (or stable step IDs) for progress UI — **next**
- [x] Docs + this checklist updated

## Phase 3 — Localhost API + WebUI

- [ ] `netductor operator serve` bind **127.0.0.1 only**
- [ ] JSON endpoints for fleet/primary/secondary + event stream
- [ ] `go:embed` minimal WebUI (Fleet form + log + credentials path)
- [ ] Security note in docs (loopback only)
- [ ] Docs + this checklist updated

## Phase 4 — Optional

- [ ] Binary split `netductor` (node) vs `netductor-op` (operator)
- [ ] Mobile day-2 client (not bootstrap)

## Done recently

- [x] v0.8.94 — legacy huh deploy removed; framed Fleet wizard
- [x] v0.8.95 — `internal/operator` use-cases; CLI/TUI thin clients; `deploy fleet`
