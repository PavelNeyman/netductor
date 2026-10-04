# Netductor line review

Method: every Go file, security first, then correctness. Marker is the resume point.
Do not mark a package done until each file in it is read. Findings stay even if fixed later.

Inventory at start: 373 Go files, ~60316 lines. VERSION 0.9.229.

## Marker

- Pass: 1
- Stopped before: `cmd/netductor/api_*.go` (except session token notes below)
- Next file: `cmd/netductor/api_session.go`
- Done packages: none fully
- Touched: `internal/session/session.go` (token storage only)

## Pass 1 — trust boundaries

Read order:

1. `cmd/netductor/api_*.go`, `serve.go` — who can call what
2. `internal/session`, `internal/mtls`, `internal/secondary`
3. `internal/policy`, `internal/vpn/apply.go`, `secondary_box.go`
4. `internal/update`, `internal/stack`, `internal/addons`
5. `cmd/netductor-tg`
6. `cmd/netductor-op`, `cmd/netductor-agent`
7. `internal/install`, `internal/edge`, `internal/nvr`
8. remainder

## Findings so far

- F1 secondary is trusted for service identity. Primary ACL on :9443 sees the UUID secondary dials, not the phone user. Documented in `secondary_box.go`. Not a bug if secondary root is equivalent to primary root.
- F2 `internal/session` stores only sha256 of the operator token, but `loadMeta` still checks a legacy path `Dir()/token` (plaintext filename). Confirm no leftover files and that new sessions never write that path.
- F3 addon update (`POST /api/addons/update`, TG button) recreates lampac/registry. Must sit behind the same operator session as other destructive routes. Verify in pass 1 on `api_addons.go`.
- F4 public Reality invalid noise was reduced by moving secondary internet uplink to 10.87.10.1:443 (0.9.227). Confirm live config after upgrade.

## Refactor backlog (fill while reading)

- R1 policy dial, sniff order, and secondary uplink tags grew inside `secondary_box.go` / `apply.go`. Split route builders when the next edit touches them.
- R2 addon versions and updater overlap (`versions.go` and `update.go`). One status struct.
- R3 catalog actions and TG callbacks duplicate labels. Prefer catalog as the only action list.

## Resume prompt

Read `docs/REVIEW-PROGRESS.md`. Continue from the marker. Do not restart. Append findings. Move the marker only after the file is read.
