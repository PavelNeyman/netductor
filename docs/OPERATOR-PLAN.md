# Operator core — implementation checklist

**Source of truth for progress.** AI agents: see mandatory update rule in [AGENTS.md](../AGENTS.md).

Legend: `[ ]` todo · `[x]` done · `[~]` in progress

## Phase 0 — Documentation

- [x] ARCHITECTURE-OPERATOR.md, compressed handoff, AGENTS discipline, this checklist

## Phase 1 — Use-case layer

- [x] `internal/operator` Specs + DeployPrimary/Secondary/CollectCredentials
- [x] CLI + TUI thin wrappers; field helpers

## Phase 2 — FleetDeploy

- [x] `FleetDeploy` / `FleetDeployWithReport`
- [x] TUI Fleet + CLI `deploy fleet`
- [x] Structured `operator.Step` events

## Phase 3 — Localhost API + WebUI

- [x] `netductor operator serve` bind **127.0.0.1 only** (default port 7373)
- [x] `POST /v1/fleet` streaming log; `GET /v1/health`
- [x] `go:embed` minimal WebUI (Fleet form)
- [x] Security: non-loopback bind rejected in code + docs
- [x] Docs + this checklist updated

## Phase 4 — Binary split

- [x] `binaryRole=node|operator` (ldflags); assets `netductor-linux-*` vs `netductor-op-*`
- [x] Deploy downloads **node** binary only onto VPS
- [x] Release workflow + CI build both surfaces
- [ ] Mobile day-2 client (not bootstrap)

## Releases

- [x] v0.8.94–0.8.97 operator program
- [x] v0.8.98 operator HTTP hardening (token, host validation)
- [x] v0.9.0 binary split operator / node
