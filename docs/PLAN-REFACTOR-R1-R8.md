# Plan: cut god functions (R-1…R-8)

**Status:** plan only. Do not start until the current fix queue is done.  
**Rule:** one slice per commit. Tests stay green. No behavior change in the same commit as a move.

## R-1 God functions — order

| Step | Target | Cut |
|--|--|--|
| 1 | `cmd/netductor/api_edge.go` | **done 0.9.433** — `registerEdgeMeta/Templates/Enroll/Storage/Agent/Admin`. Dispatcher only in `api_edge.go`. |
| 2 | `cmd/netductor-tg` `handleCallback` | **done 0.9.434** — hubs: edge approve, tools, hubs, users, nodes. Dispatcher left in `handlers_cb.go` (453 lines). |
| 3 | `handleGitCB` | **done 0.9.435** — hub, build-host, ship, proj, repo. Dispatcher 29 lines. |
| 4 | `runDoctorNative` | **done 0.9.436** — checks moved to `doctorRunChecks` via `doctorHooks`. `runDoctorNative` is setup + call. |
| 5 | `registerSessionAPI`, `registerGitAPI`, `registerNVRAPI` | Same as step 1, after edge is the template. |
| 6 | `DeployEdge` / `runDeploy` | Already stepped; split only if a step file exceeds ~200 lines. |

Stop after each step. Do not combine with i18n or auth changes.

## R-2 Duplicated helpers

Move, do not rewrite:

| Symbol | Keep in |
|--|--|
| `htmlEsc` | `internal/format` (already exists) |
| `shellQuote` | `internal/deploy` (one copy; others call it) |
| `constEq` | `internal/httpx` or new `internal/secreq` |
| `truncate` / `esc` | `internal/format` |

## R-3 `bash -c`

Replace only where the argument is data (keys, user input). Leave fixed admin strings. `wg pubkey` already moved to stdin (0.9.430).

## R-4 Timeouts (see F7)

**done 0.9.433** for node `serve` via `httpx.NewServer` (60s write). Op serve keeps 2h WriteTimeout for deploy streams. One `httpx.NewServer(addr, handler)` with ReadHeader 10s, Read 30s, Write 60s, Idle 120s. Loopback op serve can stay longer.

## R-5 Edge auth

Point tests at `DeviceIDFromAuth` + status. Delete `ValidBearer` in that same commit (it is test-only and stricter than production).

## R-6 Constants

Only when a caller is already being edited. Do not mass-rename `ndconfig.DefSvc*` vs `svcpaths.AddrP_*`.

## R-7 `serve.go`

Split jobs vs deploy vs static after the jobs API is stable. Not before.

## R-8 Test helpers

`edge.resetUSBPrevForTest` → `export_test.go` when that file is next touched.

## Not in this plan

gofmt sweep (F4), RU translation (D-1), dead-code sweep (F5) — separate docs.
