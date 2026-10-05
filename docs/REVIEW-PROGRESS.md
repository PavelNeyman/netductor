# Netductor line review

Method: every Go file, security first, then correctness. Marker is the resume point.
Do not mark a package done until each file in it is read. Findings stay even if fixed later.

Inventory at start: 373 Go files, ~60316 lines. VERSION 0.9.229.

## Marker

- Pass: 1
- Stopped before: `cmd/netductor/api_*.go` (except session token notes below)
- Next file: `cmd/netductor/api_secondary.go`
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


## Pass 1 notes (2026-10-04)

- F5 `POST /api/addons/update` had no `requireSession`. Fixed in this pass. API binds 127.0.0.1:8787 by default, so WAN could not hit it, but any local process could recreate containers.
- F6 `POST /api/stack/apply` and `/api/stack/rollback` also had no session check. Same bind. Guard added. Status route left open on purpose for local health.
- F7 `api_firewall.go` and `api_update.go` still need a line-by-line auth pass. Marker moves to secondary agent routes next, then back to those two.
- Legacy session path in `loadMeta` still unread beyond the hash note.


## Pass 1 notes (secondary/firewall/update)

- Agent routes `/api/secondary/agent/*` use device bearer token, not operator session. Backup latest and backup key are available to any valid agent token. That is the peer-restore path; a stolen secondary token can read the backup key.
- Operator secondary routes (cmd, export, exit) already call requireSession.
- F8 firewall apply/heal and update github-token/apply had no session check. Guard added this pass. Status/list left open on localhost.
- Next file: `cmd/netductor/api_edge.go` enroll/cmd routes (agent-shaped, confirm token).


## Pass 1 notes (edge)

- `/api/edge/enroll` is not open. It needs a recovery or bootstrap bearer and is rate-limited per IP. A new device lands pending and notifies TG.
- `/api/edge/commands` and `cmd_result` require the device token and reject a mismatched device_id.
- `/api/edge/mtls/material` requires the device token.
- Operator edge routes (templates, approve, rsc) use requireSession.
- Next file: `cmd/netductor/api_nvr.go` clip/token routes.


## Pass 1 notes (edge)

- `/api/edge/enroll` requires recovery or bootstrap token and is rate-limited by IP. New device stays pending and notifies Telegram.
- `/api/edge/heartbeat`, `/commands`, `/cmd_result`, `/mtls/material` take the device token and reject a mismatched device_id.
- Firewall apply/heal already require session (confirmed on disk).
- F9 `POST /api/update/apply` had no session check. Guard added this pass.
- Next file: `cmd/netductor/api_nvr.go` clip/token routes.


## Pass 1 notes (nvr)

- Camera, config, retention, PTZ, go2rtc, storage, recorder start/stop require operator session.
- Ingest accepts an approved edge device token or a session. Any approved device can upload a segment for any camera_id. Worth binding camera to site later.
- Clip download is a one-time token, max TTL 3600, path must stay under the segments root. Token is in the query string, so it can land in access logs for that one request.
- Next file: `cmd/netductor/api_vpn_http.go`.


## Pass 1 notes (vpn http)

- `/vpn/users` and `/vpn/users/{name}/...` require operator session. Create, revoke, QR, links, and policy set are behind it. Names go through `vpn.ValidName`.
- Subscription GET returns 410. Dead `subscription_legacy_disabled` still served the file to a session holder. Removed this pass.
- Backup run/list/schedule in the same file also require session.
- Next file: `cmd/netductor/api_policy.go`.


## Pass 1 notes (policy API)

- `/api/services`, delete, `/api/edge/device-policy`, and `/api/policy/apply` all require operator session. Apply is POST only.
- `SetUserPolicy` validates against the catalog after normalize. Unknown `services_mode` collapses to `list`.
- Catalog upsert from this route does not accept endpoints, so a new service cannot open a port by itself. Port rules still come from the seeded catalog.
- Next file: `cmd/netductor/api_git.go`.


## Pass 1 notes (git)

- All git routes require operator session. Pipeline name is `filepath.Base`, artifact read rejects `..`.
- F10 `git show` passed `rev` straight to git. A leading `-` could be a flag. Rejected.
- F11 workflow path could be absolute and read a file outside the checkout. Rejected.
- Pipeline scripts in the pipeline dir run as the service user. That is operator-equivalent, not a public hole.
- Next file: `cmd/netductor/api_registry.go`.


## Pass 1 notes (registry)

- status, ensure, stop, crane, auth, and catalog all require operator session. Mutations are POST.
- Registry process binds `127.0.0.1:5000` unless `NETDUCTOR_REGISTRY_ADDR` overrides it. Auth is optional and only on if htpasswd exists.
- If the addr override is not loopback and auth is off, the catalog is open on that interface. Default install is loopback.
- Next file: `cmd/netductor/api_dns.go`.


## Pass 1 notes (dns)

- list, set, and reload require operator session. Mutations are POST and audited.
- F12 `dns set` treated an unknown id as a URL and wrote it into blocky YAML. Restricted to catalog ids.
- Next file: `cmd/netductor/api_sshhosts.go`.


## Pass 1 notes (ssh hosts)

- List, forget, and clear require operator session. Clear is POST. Forget deletes a map key, not a path.
- Next file: `cmd/netductor/api_nodes.go`.


## Pass 1 notes (nodes)

- List and desired hostname require operator session. Self-register allows loopback or a session. Loopback is the peer address, not a forwarded header.
- Hostname apply uses `hostnamectl` with one argument, not a shell. Revoke session requires a valid bearer token.
- Next file: `cmd/netductor/api_svc_paths.go`.


## Pass 1 notes (svc paths)

- F13 status, apply, failover, and failover tick had no session check. Guard added. API remains localhost by default.
- Bootstrap of keys stays out of this API.
- Next file: `cmd/netductor/api_fleet.go`.


## Pass 1 notes (fleet)

- F14 `/api/fleet/digest` had no session check. Guard added. Read-only, localhost by default.
- API file pass is complete. Next package: `internal/session/session.go` (legacy token path).

## Refactor backlog added this pass

- R4 Auth is copied into each handler. That is why addon update, stack apply, svc-paths, and fleet digest shipped open. One wrapper for operator routes, with an allowlist for agent and health, would stop the misses.
- R5 `register*API` is split across 18 files with no shared route table. Catalog already lists actions. Generating or checking routes against the catalog would catch a handler that forgot the session check.
- R6 Status routes (`stack`, `update`, `firewall`) are still open on localhost while their mutations are guarded. Decide one rule: all `/api` except agent and `/health` require a session.


## Pass 1 notes (session)

- New sessions store only sha256 in `<hash>.json`, mode 0600. Token is 32 random bytes. Hours are clamped to 72.
- F2 `loadMeta` still accepted a legacy file whose name was the raw token. Removed. Revoke still deletes that leftover name.
- Cookie `nd_session` is accepted as a bearer equivalent. Flag and Secure are set by the caller, not here.
- R7 two session formats in one lookup. Now one. Next file: `internal/mtls`.


## Pass 1 notes (mtls)

- Agent plane requires a client cert, TLS 1.3, and checks the serial against the revoke list. Node id is sanitized before it becomes a directory name.
- Any cert from this CA is accepted. The callback does not bind the cert CN to a node id. Revocation is the cut-off. That matches a private CA, not per-node pin.
- R8 `VerifyPeerCertificate` only checks revoke. A later pass can require the cert subject to match a known node if we want stolen-but-not-revoked certs to fail closed.
- Next file: `internal/secondary`.
