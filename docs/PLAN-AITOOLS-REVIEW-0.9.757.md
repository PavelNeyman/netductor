# Plan: findings from ai-tools review (netductor 0.9.757)

**Status:** open · **no release until closed or explicitly deferred**  
**Source:** `aitools v1.12.0` review on `main` @ `37f6dac` / VERSION **0.9.757**  
**Artifacts:** sandbox `artifacts/aitools-report-netductor-0.9.757.{json,txt}`  
**Rule:** fix in code first; cut release only when agreed. Migrations/floors per CF-15 / `AGENTS.md`.

EN · [RU](ru/PLAN-AITOOLS-REVIEW-0.9.757.md)

---

## Scope of the scan

| Tool (via ai-tools) | Result |
|---------------------|--------|
| gosec | Ran; **1213** issues (mostly noise G204/G304) |
| staticcheck | Ran; useful signal (SA4023, U1000, style) |
| deadcode / errcheck / ineffassign / govulncheck | **Broken in aitools ≤1.12.0** (`"."` instead of `./...`) — fixed in **ai-tools v1.12.1**; re-run after that release |
| trivy / shellcheck / golangci-lint | Not installed in that run |

Re-run full review after ai-tools **≥ v1.12.1** before treating deadcode/errcheck as empty.

---

## Priority queue

### P1 — fix first

| ID | Finding | Location | Action |
|----|---------|----------|--------|
| **N1** | **SA4023** — comparison always true; `sessionOnce` never returns a nil interface | `internal/revbridge/client.go` ~52 / ~67 | Redesign error return (concrete `error` + ok, or sentinel); do not treat “nil iface” as failure. Add unit test. |
| **N2** | **G123** — `tls.Config` uses `VerifyPeerCertificate` while session resumption may skip it; `VerifyConnection` not set | `internal/mtls/mtls.go` | Align with current Go TLS guidance: disable tickets if relying only on VerifyPeerCertificate, or set `VerifyConnection`. Document trust boundary. Test if feasible. |

### P2 — cleanup / hygiene

| ID | Finding | Location | Action |
|----|---------|----------|--------|
| **N3** | **U1000** unused symbols | `internal/devices` (`reFrom`), `internal/notify` (`lastFlush`), `internal/nvr` (`dirSizeBytes`), `internal/update` (`isChecksumish`, `holdHostLock`), `internal/vpscheck` (`runCmd`), op TUI styles/menu leftovers | Remove or wire; prefer delete dead paths |
| **N4** | Style staticcheck | S1008/S1024/S1017/S1000, ST1005 error strings, ST1013 HTTP status literals | Batch cosmetic pass; no behavior change |
| **N5** | **G115** integer conversion | firewall, nvr, revbridge, svcpaths, tapo | Review bounds; cast safely where values can exceed target type |

### P3 — known / intentional (do not “fix” blindly)

| ID | Finding | Notes |
|----|---------|--------|
| **N6** | G204/G304 mass | Expected on SSH/deploy/`sshpass`/provision; keep review on **injection** (unquoted host/cmd), not on exec existence |
| **N7** | G402 InsecureSkipVerify | vpscheck DPI, sni_health, tapo, probes — intentional for probe/vendor; must **not** appear on user VPN / mTLS control plane |
| **N8** | G401/G501/G505 SHA1/MD5 | Tapo / ONVIF / WS accept — vendor protocol |
| **N9** | G101 “hardcoded credentials” | False positives on path constants (`telegram_bot_token` path, i18n, widget JS) |

---

## Process

1. Implement **N1 → N2 → N3** (and N4/N5 as bandwidth allows).
2. Re-run **ai-tools ≥ 1.12.1** on tree; attach new report under `docs/audits/` or artifacts note in handoff.
3. **No** netductor GitHub release until this plan is updated with `[x]` evidence (commit SHA + test names) or items explicitly deferred here.
4. If a fix changes on-disk format / apply order / peer material → `upgrade.DeclareBinaryFloor` in the **same** commit (CF-15).

---

## Out of scope (this plan)

- Fleet mesh Phase 5 multi-master  
- Hardware e2e OpenWrt / NVR  
- Full gosec zero-issue goal  

## History

| Date | Event |
|------|--------|
| 2026-11-18 | Initial plan from aitools 1.12.0 pass on 0.9.757 |

## 2026-11-18 re-run (ai-tools 1.12.1-dev local)

- **`no Go files` gone** with `./...` args.
- **errcheck**: ~330 findings (mostly deferred Close/Remove — triage later).
- **ineffassign**: ~37 (TG keyboards, mtls rollover, vpn apply…).
- **deadcode / govulncheck**: `signal: killed` (sandbox OOM on full tree) — re-run on larger host or narrower package set.
- **SA4023** confirmed again via staticcheck.

Artifact: `artifacts/aitools-report-netductor-0.9.757-r2.*`


## Progress 2026-11-18

| ID | Status | Evidence |
|----|--------|----------|
| **N1** | **done** | `sessionOnce` returns `nil` on normal end, `errClientStopped` on Stop; loopKind distinguishes. Test `TestSessionOnceReturnsNilInterfacePossible`. staticcheck SA4023 cleared on package. |
| **N2** | **done** | `ServerTLSConfig`: `VerifyConnection` + `SessionTicketsDisabled` (revoke on every conn). |
| **N3** | **partial** | Removed unused `reFrom`, `dirSizeBytes`, `lastFlush`, `isChecksumish`, `holdHostLock`, `runCmd`. op TUI U1000 still open. |

## Progress 2026-11-18 (pass 2)

| ID | Status |
|----|--------|
| **N1** | done (prior) |
| **N2** | done (prior) |
| **N3 UI/cmd** | **done** — op TUI dead styles/menuItem/list/capture; cli_flags helpers; TG widget dead helpers; formatClusterHTML; lastAccessMsg; thin wrappers applyUFW/installHealthTimer/recoverPrevSnapshotSwap |
| **N4** | partial — mux S1000 fixed |
| **N3 internal residual** | **deferred** — ~20 U1000 in deploy/firewall/install/svcpaths/secondary (need wire-or-delete audit, not blind delete) |
| **N5 G115** | deferred — needs per-site bounds review |

`staticcheck -checks U1000 ./cmd/...` = 0. Full `./...` residual listed under deferred.

## G115 pass 2026-11-18

Journal: [audits/2026-11-18-G115-integer-overflow.md](audits/2026-11-18-G115-integer-overflow.md) (RU: [ru/audits/…](ru/audits/2026-11-18-G115-integer-overflow.md)).

- [x] Site-by-site classification (43 findings)
- [x] Real: ws frame DoS bound; nvr Statfs math; euid compare without uint32 narrow
- [x] safe-bound: firewall hostU32, backup chunk len helper
- [x] Re-run ai-tools/gosec for delta count (G115 43→20, G123=0; tip 9205bbe)
- [ ] U1000 residual internal (separate)

