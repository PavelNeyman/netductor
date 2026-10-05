# Netductor line review

Method: every Go file, security first, then correctness. Marker is the resume point.
Do not mark a package done until each file in it is read. Findings stay even if fixed later.

Inventory at start: 373 Go files, ~60316 lines. VERSION 0.9.229.

## Marker

- Pass: 1
- Stopped before: `cmd/netductor-op` (operator Web/TUI backend)
- Next file: `cmd/netductor-op/main.go` then `internal/operator/`
- Done packages: API handlers, session, mtls, secondary agent, policy, vpn, update, stack, addons, **cmd/netductor-tg** (all handlers)
- Last fix: F18 ScheduleApply shell injection → 0.9.246


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


## Pass 1 notes (secondary agent)

- Command switch is an allowlist: reboot, upgrade, journal, backup, mtls refresh, restart of named units. Unknown commands are rejected.
- F15 `upgrade:<tag>` was concatenated into the GitHub URL. A tag with `/` could leave the release path. Tag is now digits and dots only.
- Agent token is sent as Bearer. Backup key fetch still uses that token (noted earlier).
- R9 command dispatch is a growing switch. A table of name to handler would make the allowlist obvious.
- Next file: `internal/policy` (routes already touched; read remaining files).


## Pass 1 notes (policy package)

- Validate rejects unknown service ids unless mode is all. Normalize collapses any other mode to list, so a bad mode cannot become all.
- Presets: full = all internal, media = lampac+nvr, none = internet only. Unknown preset leaves the current policy.
- Integrity check warns on internal services with no port. It does not block apply.
- R10 media preset lists nvr even when recording is off. Route generation skips it via ServiceEnabled, so the card and the sing-box rule can disagree. One gate should feed both.
- Next file: `internal/vpn/secondary_box.go` uplink and identity.


## Pass 1 notes (secondary_box)

- Internet uplink dials 10.87.10.1:443 as relay-uplink, with mux and no vision. Service net dials 10.87.10.1:9443 as the real user UUID, no mux. Unmatched 10.88.0.0/24 is blocked.
- Tags are sanitized to letters and digits. User name goes into JSON, not a shell string.
- GeoIP rule-set is downloaded direct from GitHub, not through the Reality uplink. A bad rule-set can change RU split. Expected, but it is a supply-chain input.
- R11 secondary still sniffs before the 10.88 rules. Primary already moved service rules above sniff. Same order here would keep service dials from waiting on sniff.
- Next file: `internal/vpn/apply.go`.


## Pass 1 notes (apply.go)

- Public inbound listens on `::` and the VLESS port, mux on, vision for users, no vision for relay-uplink. Service inbound listens only on 10.87.10.1:9443, no mux, no vision.
- Service rules are built before sniff. Disabled users are omitted. Config is written 0600 via a temp file.
- vless-svc still lists every enabled user, including relay-uplink. ACL, not the inbound user list, is what blocks services. A relay-uplink connection to :9443 is accepted and then rejected by route.
- R1 still stands: inbound build, route build, and exit outbound share this file. Split when the next edit touches it.
- Next file: `internal/vpn/registry.go`.


## Pass 1 notes (registry)

- Users file and link files are written 0600 via a temp file and rename. UUID comes from sing-box or kernel random. Name must match `^[a-zA-Z0-9_][a-zA-Z0-9_-]{0,63}$`.
- Add writes the registry before artifacts. A failed artifact write leaves a user without links. Not a security hole.
- Subscription files are still written (`subscription-full.txt`, `qr-subscription.png`) even though the API returns 410. Operator-only on disk.
- R12 stop writing subscription artifacts, or the 410 story and the disk disagree.
- Next file: `internal/vpn/users.go`.


## Pass 1 notes (users.go)

- Add, note, disable, enable, and revoke go through native registry helpers. `run` is a fallback that execs the same binary with separate args, not a shell.
- F16 `ReadClient` joined the filename without checking it. A `../` candidate could leave the client dir. Name must now be valid, and the file name cannot contain a slash.
- R13 native-then-fallback can hide a real registry error and try the CLI path. Prefer one path.
- Next file: `internal/update`.


## Pass 1 notes (update)

- Download verifies SHA256SUMS unless NETDUCTOR_UPDATE_SKIP_VERIFY=1. Missing sums abort. Mismatch deletes the temp file.
- Asset name is chosen from a switch, not from the request. Repo is a constant.
- F17 release tag was concatenated into the GitHub URL. Same class as the secondary upgrade tag. Tag is now digits and dots.
- R14 skip-verify is an env flag on the host. A local process can set it. Worth requiring the operator session path to refuse the flag.
- Next file: `internal/stack`.


## Pass 1 notes (stack)

- Apply refuses a tag older than the running version, takes a lock, deletes attempt/, and does not auto-downgrade on health fail. New binaries stay, prev/ is not updated.
- Pin blocks apply and rollback. Secondary is not upgraded by stack apply.
- Health fail returns an error after the binary is already replaced. The operator sees failure while the new file is live. Intentional after the 121/132 loop, but the message must stay explicit.
- R15 health check sleeps in the apply process. A wrapper that reports "installed, health pending" would avoid a long API call looking like a hang.
- Next file: `internal/addons`.


## Pass 1 notes (addons)

- `lampac.go` probes docker inspect/stats and loopback HTTP only. Container name is fixed (`netductor-lampac`). No operator input in paths.
- `Update(name)` only runs registered updaters (`lampac`, `registry`, `git`) or `all`. Unknown names error. API/TG already session-gated (0.9.230).
- `updateLampac` / `updateRegistry` pull fixed image refs then reinstall. Supply chain is GHCR/`registry:2`, same class as GeoIP rule-set.
- `updateGit` runs `apt-get install -y git`. Operator-equivalent on the host.
- `readTag` joins `EtcDir()/addons/` + name + `.tag`. Call sites only pass registered names. No traversal from operator input today.
- R2 confirmed: `Versions()` embeds `All()` and a separate `lampac` map; `ListAddons` also reports lampac. One status shape later.
- Next: `cmd/netductor-tg` (admin gate, then destructive handlers).


## Pass 1 notes (stack schedule — found while leaving addons)

- F18 `ScheduleApply` built `bash -c` with the release tag interpolated (`stack apply %s`). A tag with shell metacharacters became command injection. TG `m:updates:apply:` and API schedule feed this path.
- Fixed: `update.ValidReleaseTag` (digits and dots only, leading `v`), and the tag is passed as argv `$1`, not concatenated into the script text.


## Pass 1 notes (tg — start)

- Admin id from secrets file or `NETDUCTOR_TG_ADMIN`. First-message claim only if `NETDUCTOR_TG_CLAIM_FIRST=1`.
- Callbacks: `cq.From.ID != admin` ignored. Messages: `m.Chat.ID != admin` return.
- Destructive actions (stack, firewall, backup, edge) go through `exec.Command` with fixed binaries/args after the admin gate.
- Next file: rest of `cmd/netductor-tg` handlers (certs, edge guest, policy set from buttons).


## Pass 1 notes (telegram bot)

Files read: `main.go` (admin gate, long-poll), `handlers_cb.go`, `handlers_msg.go`, `handlers_extra.go`, `handlers_versions.go`, `handlers_stack.go`, `handlers_certs.go`, `handlers_mtls.go`, `handlers_policy.go`, `handlers_git.go`, `handlers_registry.go`, `handlers_nvr.go`, `handlers_edge_guest.go`, `handlers_edge_template.go`, `handlers_edge_luci.go`, `catalog_ui.go`, `format_*.go`, `keyboards.go`, `i18n.go`.

### Trust

- Admin id from file / `NETDUCTOR_TG_ADMIN`. Callbacks require `cq.From.ID == admin`. Messages require `m.Chat.ID == admin`.
- Claim only if `NETDUCTOR_TG_CLAIM_FIRST=1` on `/start`. Default: no open claim.
- Destructive work uses `exec.Command(netductorBin(), args...)` (argv), not shell — except F18 which was fixed in 0.9.246 (`ScheduleApply` + `ValidReleaseTag`).

### Destructive operator power (by design)

- Node card: reboot / upgrade secondary via agent allowlist; primary `local-cmd reboot|upgrade` can reboot this VPS or run apt + wget of **embedded** `deploy.Release` binaries (not latest GH tag, no SHA in that path). Operator-only.
- Stack heal / rollback / schedule apply from TG.
- Secondary provision: password is a CLI argv briefly (visible in process list to root on the same host).

### Catalog

- `m:op:<id>` only runs IDs present in `opcatalog`. Session is created for localhost API and revoked after the request.
- Edge `EnqueueCmd` rejects unknown actions (`allowedEdgeActions`); guest_status/grant/revoke allowed.

### No new open hole this pass

- F18 already closed.
- No additional public or non-admin injection found in TG handlers.

### Refactor backlog (append)

- R4 primary `nodes local-cmd upgrade` should reuse `stack`/`update` download+SHA path instead of wget of fixed Release.
- R3 still: prefer catalog for all TG destructive buttons where possible.
- R5 secondary provision password: prefer SSH key or env-file for sshpass, not argv.
