# Secondary Deployment Safety Remediation Plan

Base reviewed: `main` at `fc745c0564e4dd82ad8838687d7a564cb1b6c543`.

This plan targets the current source tree and the next published operator/server release. The installed operator version is not a design constraint. The upgrade path from already-running Netductor nodes is a first-class requirement.

## Priority 0 — preserve management access during SSH hardening

- Make rollback independent from a potentially stalled prepare/finalize command.
- Define one explicit transaction state machine for prepare, key-only verification, finalize, cancel, and rollback.
- Bound every lock acquisition and every systemd stop/start operation.
- Ensure rollback cannot be permanently blocked by a process holding the transaction lock; ensure finalize cannot continue mutating SSH/firewall after rollback has claimed the transaction.
- Validate `sshd -t`, listener state, firewall state, and key-only reachability before removing the factory SSH path.
- Keep the previous SSH/firewall snapshot until the transaction reaches a verified terminal state.
- Add shell-level fault-injection tests for lock contention, hung finalize, rollback/cancel races, failed sshd restart, and failed firewall restore.

Acceptance: every failure before verified completion either preserves the old access path or leaves a deterministic, documented recovery path; no silent success.

## Priority 1 — make bootstrap rollback cover runtime state

- Extend the bootstrap journal to snapshot the active/enabled/masked state of every unit the transaction may change, before the first service mutation.
- On rollback, restore files, run daemon-reload, then restore the recorded unit state in a deterministic order.
- Treat failure to restore any unit as rollback failure; retain the journal for retry/recovery.
- Ensure a crash during rollback is recoverable and repeated recovery is idempotent.
- Preserve compatibility with existing version-1 file-only journals. Journal schema changes must be versioned and old in-flight journals must remain recoverable.
- Add tests for units initially active, inactive, enabled, disabled, masked, absent, and for failures halfway through enable/restart.

Acceptance: after a failed bootstrap, disk configuration and systemd runtime/enablement state match the pre-transaction snapshot, or the transaction remains explicitly failed and recoverable.

## Priority 2 — firewall transaction and safe migrations

- Snapshot only the exact Netductor-managed firewall rules that the operation may mutate; never snapshot/restore the entire administrator firewall policy as an opaque overwrite.
- Record before/after rule deltas and reverse only this transaction's changes.
- Reconcile legacy global WSS rules by first ensuring source-restricted rules for every existing peer, then removing only tagged legacy rules.
- Re-read numbered UFW rules after every mutation that can shift rule numbers.
- Detect conflicting administrator rules (including the existing Secondary `80/tcp ALLOW`) and stop before destructive changes. Do not silently delete or reorder administrator rules.
- Make migrations idempotent, restart-safe, and explicit about manual conflicts.

Acceptance: adding the new Secondary cannot strand the existing peer or broaden ingress; rerunning migration produces no additional changes.

## Priority 3 — upgrade compatibility and migration framework

- Inventory all persisted formats and host-level state touched by the current bootstrap/update paths: material, peer registry, operation journals, policy, secrets, systemd units, SSH drop-ins, and firewall rules.
- Add explicit schema/version detection and migrations where a format or invariant changes.
- Migrations must be idempotent, write-ahead journaled where they mutate multiple resources, and recoverable after process/host interruption.
- Never infer that a missing file means a clean installation when a legacy representation may exist.
- Verify operator/server version negotiation and ensure the operator does not start a deployment with an incompatible remote payload.

Acceptance: upgrade from the currently deployed generation to the new release preserves peer identity, keys, routes, service state, and management access; interrupted upgrades can be safely resumed.

## Priority 4 — end-to-end Secondary deployment readiness

- Trace the complete operator -> Primary peer allocation -> remote Secondary bootstrap -> validation -> ready/promote path.
- Verify slot allocation and registry locking, WSS source restrictions, WG addressing, service units, health checks, and rollback/recovery.
- Explicitly handle the known new-host firewall conflict on `80/tcp` as a preflight failure requiring deliberate operator action; do not remove it automatically.
- Keep the current Secondary active until the new peer has passed validation and an explicit promotion is performed.
- Do not deploy or alter production servers as part of repository changes.

Acceptance: a documented, repeatable deployment sequence can add the new Secondary without interrupting the existing peer, and promotion remains a separate reversible operation.

## Priority 5 — verification and release

- Add unit and fault-injection coverage for each acceptance criterion.
- Run focused Go tests locally/through an explicitly requested manual workflow; do not re-enable push/PR test triggers.
- Run the full manual test workflow before release.
- Review the complete diff for API, CLI, config-format, migration, and rollback regressions.
- Publish a versioned release only after verification; update the operator and server payloads together.

## Execution order

1. SSH rollback state machine and fault-injection tests.
2. Bootstrap systemd runtime-state journal and recovery tests.
3. Firewall delta journal and migration tests.
4. Upgrade/migration inventory and compatibility tests.
5. End-to-end Secondary deploy-path audit and release preparation.

Production deployment remains a separate, explicitly authorized step.

## Implementation status

Updated against current `main` after the initial plan commit.

- [x] SSH rollback lock acquisition is bounded. On contention, the rollback attempt adds tagged emergency management ingress, records `ROLLBACK_FAILED`, and exits non-zero; systemd retries the rollback service until the transaction lock clears. The emergency iptables rules are tagged and removed only after the original firewall state is restored successfully.
- [x] Bootstrap journal schema is now version 2 and records systemd load, enablement, and active state before service changes. Version-1 journals remain accepted and recoverable.
- [x] Bootstrap records per-unit mutation intent durably immediately before each service mutation. Rollback restores only units that the transaction may have touched, after restoring files and running daemon-reload.
- [x] Added focused tests for bounded SSH rollback/retry, systemd state restoration, absent units, atomic unit snapshots, untouched-unit preservation, and version-1 journal compatibility.
- [ ] Focused tests have **not** been executed yet. Actions remain manual-only; no workflow was dispatched.
- [x] Bootstrap journal now records exact before-images for transaction-created source-restricted WSS/service peer rules on both UFW and iptables. Recovery removes only rules absent before the transaction; pre-existing rules and unrelated administrator rules are preserved. UFW migration rules for existing peers remain additive/restrictive and are deliberately not rolled back.
- [x] Firewall mutation lock acquisition now has a 30-second bounded variant, used by Primary bootstrap and rollback.
- [x] Secondary WSS bootstrap now also uses the 30-second bounded firewall lock; lock contention fails bootstrap cleanly instead of hanging indefinitely.
- [x] Added focused UFW rollback tests proving that only the transaction-created peer rule is removed and a pre-existing peer rule is retained.
- [ ] Focused tests have not been executed yet. The repository changes are source-reviewed only; no Actions workflow was dispatched.

## Upgrade compatibility audit — implementation update

- [x] Versioned the state-repair completion marker. Older installations contain the timeless `ok` marker; the new `v2` marker forces exactly one additional repair pass so files and invariants added to `RepairStateOnUpgrade` in later releases are not permanently skipped.
- [x] Added a regression test proving that an old `ok` marker triggers the new repair generation once, then becomes idempotent.
- [x] Peer registry loading now normalizes the original unversioned representation (version 0) to schema 1, but rejects unknown future schema versions. Saving also rejects unsupported versions instead of silently rewriting them.
- [x] Added tests for legacy unversioned peer registry compatibility and fail-closed behavior for schema 2.
- [ ] Remaining persisted formats and all version-negotiation call sites still require review. These changes have not been test-executed.


### Legacy peer-material import guard
- The legacy root `material.json` import now validates its tunnel identity before normalization and refuses any material declaring a non-zero slot. This prevents an existing slot-1+ record from being silently rewritten as slot 0.
- Existing legacy slot-0 material remains compatible, including keys-only legacy records whose slot-0 tunnel values are absent.
- Added regression coverage proving a slot-1 root material is rejected without creating a peer registry entry or peer material file.
- Tests have not been executed.


### VPN user registry corruption guard
- `loadRegistry` now validates that `vpn-users.json` contains a real `users` array before decoding into the typed registry. Missing, null, or non-array values fail closed instead of becoming an empty list and being overwritten by the next user mutation.
- Added regression coverage for missing/null/object `users` envelopes. Existing absent-file behavior remains clean-install compatible.
- Tests have not been executed.


### SSH hardening lock bound audit
- [x] Bounded SSH Prepare locks (permanent, transaction, firewall), Finalize locks, CancelRollback locks, and HostAgentPurge firewall lock. Lock contention now returns an explicit error instead of indefinitely blocking the deploy workflow.
- [ ] Manual/focused tests still pending; Actions remain manual-only.

### Pending Secondary pack identity isolation

- [x] `IssueToken` now replaces only never-joined records with the same logical device name. Preparing a pack for another Secondary no longer deletes unrelated pending identities/tokens.
- [x] `deploy secondary` supplies a deterministic host-scoped pack name, so concurrent deployments to different Secondary hosts cannot replace each other's pending token.
- [x] Heartbeat pruning applies the same name-scoped rule; the first Secondary to join no longer invalidates other prepared packs.
- [x] Added regression coverage: preparing `secondary-a` again and accepting its first heartbeat both preserve the pending token and identity for `secondary-b`.
- [ ] Focused tests have not been executed; Actions remain manual-only.

### Interrupted peer allocation recovery

- [x] `peer recover` can regenerate material only when the durable operation phase is `allocating` and the material file is genuinely absent. It reuses the journaled peer ID and slot, verifies the slot/IP are not claimed, persists material atomically, then advances the journal before bootstrap.
- [x] Existing but corrupt/incomplete material, a missing material file in any later phase, or conflicting registry state remains fail-closed; keys are never silently replaced.
- [x] Added regression tests for safe regeneration and refusal to overwrite corrupt material.
- [ ] Focused tests have not been executed; Actions remain manual-only.

### Peer registry lock bound audit

- [x] Primary service-peer registry mutations now use a non-blocking flock retry loop with a 30-second deadline. A stuck peer lifecycle process returns an explicit timeout instead of hanging Secondary deployment indefinitely.
- [x] Added focused lock-contention regression coverage with a shortened test timeout.
- [ ] Focused tests have not been executed; Actions remain manual-only.

### Deploy orchestration lock audit
- [x] Primary peer firewall mutation in `deploy secondary` now uses a 30-second bounded lock; contention exits before UFW mutation rather than hanging the operator workflow.
- [ ] Deploy resume/recovery review and focused tests remain pending.

### Provisioning recovery checkpoint
- [x] Deploy journal now checkpoints `provisioned` immediately after successful `ProvisionFromCore` (which includes hardened SSH verification and rollback cancellation), before the separate operator-side :52222 probe and subsequent Primary sync. A failure in those later steps therefore resumes in key-only mode instead of incorrectly retrying initial password provisioning against an already-hardened host.
- [ ] Focused tests still pending.


### Bootstrap rollback compensation

- [x] Rollback now attempts file restoration, systemd daemon reload/runtime+enablement restoration, and firewall delta restoration independently. A failure in one domain no longer skips compensation in the others.
- [x] The transaction journal remains in `rolling_back` unless all compensation steps succeed; recovery can retry the complete rollback idempotently.
- [x] Added a regression test injecting file-restore failure and asserting systemd + firewall compensation still run and the journal remains `rolling_back`.
- [ ] Focused rollback failure-injection tests have not been executed; Actions remain manual-only.


### Secondary deploy journal host binding

- [x] Resume rejects a journal whose embedded host differs from the requested target host; prevents copied/misplaced state from reusing peer/agent identity across targets.
- [x] Added regression test for cross-host journal rejection.
- [ ] Tests not executed (manual-only Actions).

### Secondary agent mTLS identity binding

- [x] Agent endpoints now require the TLS client certificate Common Name to equal `netductor-agent-<agent_id>` resolved from the Bearer token. Token-only identity is no longer sufficient.
- [x] Heartbeat authenticates the certificate/token pair before persisting metrics, commands, recovery fingerprints, or online state.
- [x] Config, backup, backup-key, and client-material endpoints use the same certificate-to-agent binding.
- [x] Added regression coverage for absent TLS, absent peer certificate, matching identity, another Secondary identity, and the shared default client certificate.
- [ ] Focused tests have not been executed; Actions remain manual-only.

### Secondary API credential disclosure

- [x] The authenticated Secondary device and status endpoints now serialize an explicit API view that omits the stored agent Bearer token.
- [x] The API view applies the existing authorized-keys log redaction before returning the last command log.
- [x] Added regression coverage asserting that agent tokens and raw authorized-keys material are absent while device identity remains available.
- [ ] Focused tests have not been executed; Actions remain manual-only.

### mTLS client certificate rotation recovery

- [x] Rotation stages the existing certificate/key pair before issuing replacements and restores it when issuance, metadata loading, or pending-journal persistence fails.
- [x] Retries are idempotent while a rotation is pending; they return the same active replacement material instead of creating a chain of unconfirmed certificates.
- [x] Interrupted rotations recover staged files automatically: without a committed pending record the old pair is restored; with a matching pending record the replacement pair is retained and stale backups are cleaned.
- [x] A visible pending journal is never rolled back to the old pair merely because the final directory-sync confirmation failed.
- [x] Grace-expiry revocation failures and confirmation revocation failures are propagated; pending state is not silently discarded.
- [x] Added regression tests for idempotent rotation/confirmation, failed revocation, grace-expiry failure, and recovery after interruption.
- [ ] Focused tests have not been executed; Actions remain manual-only.

### CA rollover safety containment

- [x] CA rollover journal loading distinguishes a genuinely absent journal from malformed JSON and I/O errors; invalid phases and incomplete state fail closed.
- [x] New-CA issuance validates the journal and CA key pair, stages the previous client identity, and retains the old serial until the Secondary or Edge agent presents the new certificate over TLS.
- [x] Secondary and Edge agents validate the CA bundle, certificate/key pair, client-auth chain, and certificate CN before installing staged material. Both preserve unrelated files and retain rollback material until the Primary confirms the active TLS identity.
- [x] Agent rollback is handshake-aware: failed TLS handshakes can restore the old material, but a successful TLS handshake followed by a lost/error HTTP response does not revert to a certificate that may already have been revoked.
- [x] Primary derives peer serials from authenticated TLS connections, binds Edge TLS CN to the bearer-token device ID, and persists server-observed identity rather than trusting JSON serial fields.
- [x] Adoption confirmation and old-serial revocation are durable and retry-safe. The response acknowledges material only when the observed serial matches the currently active server-side certificate.
- [x] CLI promotion checks the latest server-observed TLS serial for every registered Secondary and approved Edge agent. Missing, stale, or mismatched identities block promotion.
- [x] Promotion verifies the exact new CA, every active new-CA client certificate, the previous client backups, and the new server/default-client certificate chains before committing.
- [x] CA, server, and default client certificate promotion uses a durable `promoting` journal and before-image backups. `EnsureAll` restores the previous complete set after an interrupted promotion; retries are safe.
- [x] Server mTLS configuration reloads the server certificate and client trust roots for each new TLS handshake, so promotion does not require restarting the listener.
- [x] Backups and temporary CA files are cleaned only after the `done` journal is durable. Abort refuses to discard a new CA after any client identity has been issued or adopted.
- [x] Added regression coverage for CA adoption, revocation failures, promotion recovery, Edge material validation, and Edge directory rollback.
- [ ] Focused tests have not been executed. Actions remain manual-only.

### CA rollover adoption protocol

- [x] Primary records the client certificate serial observed in authenticated TLS heartbeat; it is derived from the TLS peer certificate, never accepted from heartbeat JSON.
- [x] Rollover journal records old/new serial per issued Secondary. Adoption requires exact match with the server-observed new serial.
- [x] Old client serial is revoked only after durable adoption confirmation. Confirmation and revocation are retry-safe.
- [x] Revocation storage uses the shared durable JSON writer and an inter-process lock for read-modify-write, preventing lost updates between concurrent Netductor processes.
- [x] Abort refuses to delete the new CA after a new-CA client identity has been issued.
- [x] Finish checks issued-client confirmation, old-serial revocation, and absence of active per-node certificates signed by the old CA.
- [x] The CLI finish path checks every registered Secondary and approved Edge agent against its latest server-observed TLS serial; a missing or mismatched identity blocks CA promotion even if local certificate files are missing.
- [x] Secondary validates the CA bundle, certificate/key pair, client-auth chain, and certificate CN before installation.
- [x] Edge agents now use the same staged/validated dual-directory material replacement, switch their in-memory HTTP client after installation, and retain rollback material until the Primary confirms the observed TLS identity.
- [x] Edge heartbeat, command polling, command-result, and mTLS-material endpoints bind TLS client CN to the bearer-token device ID; rotation confirmation uses the TLS peer serial rather than JSON input.
- [x] Secondary stages and swaps the mTLS directory with recovery support, preserves unrelated files, and retains the old directory until Primary confirms the new TLS identity.
- [x] Secondary restores previous material after repeated failed heartbeat attempts with the new identity.
- [x] Edge and Secondary agents distinguish TLS-handshake failure from a lost/error HTTP response; they do not roll back after the Primary has accepted the new TLS identity, preventing reversion to a certificate that may already have been revoked.
- [x] Added focused regression tests for adoption, revocation, abort, material validation, and directory preservation.
- [x] CA, server, and default client certificate promotion now uses a durable `promoting` journal and before-image backups. `EnsureAll` rolls back an interrupted promotion before starting mTLS services; the operation can then be retried.
- [x] The promoted server and default client certificates are re-issued under the new CA. Backups and temporary CA files are removed only after the `done` journal is durable.
- [x] The mTLS server reloads its server certificate and client trust roots for each new TLS handshake, so CA promotion does not depend on restarting the listener process.
- [x] Added a recovery test that simulates a crash after only the CA certificate was replaced and verifies restoration of the CA, server, and default client material.
- [ ] Tests have not been executed. Actions remain manual-only.

### Deploy retry after committed SSH hardening

- [x] A post-hardening host-agent purge failure no longer reports Secondary provisioning as failed after SSH access has already been finalized and the rollback timer cancelled. The purge is non-fatal housekeeping and is retained as a warning in the provision result; the operator journal can safely checkpoint `provisioned` and continue on key-only SSH.
- [ ] Focused tests and full deploy-path verification remain pending. No production deployment performed.


### Secondary deploy preflight hardening

- [x] Before `prepare-pack` or any SSH provisioning/hardening of the new host, the operator reads the Primary service-peer registry and validates the requested peer ID/IP binding and slot tuple.
- [x] A new peer is rejected before touching the host when no service-plane slot is free.
- [x] Existing matching peer registration is accepted idempotently; conflicting ID/IP or inconsistent registered slot fails closed.
- [x] Added focused unit coverage for empty/full registry, matching peer, ID/IP conflicts, and invalid slot tuple.
- [ ] Tests have not been executed. Actions remain manual-only.


### Secondary UFW conflict preflight ordering

- [x] Secondary role application now detects existing ALLOW rules on restricted ports (including the known factory `80/tcp ALLOW`) before any WSS peer-rule migration or numbered-rule deletion.
- [x] The check fails closed when numbered UFW rules cannot be completely parsed; it does not delete or reorder administrator rules.
- [x] When UFW is inactive (and hides its numbered policy), the preflight also inspects `ufw show added` and rejects saved restricted-port ALLOW commands before mutation.
- [x] Added focused tests for active/inactive UFW, restricted ports, and incomplete rule parsing.
- [ ] Tests have not been executed. Actions remain manual-only.


### Registry and saved-firewall edge cases

- [x] Deployment preflight rejects duplicate active peer IDs and public IPs, even if the requested peer itself would otherwise match.
- [x] Inactive UFW saved-rule inspection also recognizes source-specific commands using `to any port N`, not only global `ufw allow N/tcp`.
- [x] Focused tests cover duplicate peer IDs and source-specific port-80 ALLOW.
- [ ] Tests have not been executed. Actions remain manual-only.

### Provisioning checkpoint-gap recovery

- Persist the Primary-issued `AgentID` in the local deploy journal before invoking Secondary provisioning, while retaining `current_step=provision_started`.
- If the process later resumes with that in-progress marker but without a successful `provisioned` checkpoint, do not repeat provisioning. Require key-only SSH on `:52222` and verify both `netductor-secondary-agent` and `sing-box` are active; only then promote the local journal to `provisioned` and continue.
- If that proof fails, fail closed and preserve the journal for operator reconciliation. This avoids a second password/bootstrap install after a possibly successful first run.
- Added a unit test for the journal predicate. Runtime recovery and all Go tests remain unexecuted; tests are intentionally manual-only.


### Stale peer-add recovery conflicts

- `peer recover` now checks the current registry before resuming bootstrap/index writes. It refuses to resurrect an identity already marked `removed`, and refuses to reuse an IP or service-plane slot that has since been assigned to another active peer.
- Added regression tests for removed identity, reassigned IP, and reassigned slot. Tests remain manual-only and were not run.

### UFW idempotency on re-deploy (observed 2026-11-04, host 195.140.147.20) — not fixed yet

Code is intentionally unchanged. Recorded so the next hardening pass does not rediscover it.

Prepare (`internal/hardening/ssh_dropin.go`):

- If `:22` is listening, prepare writes `factory-ssh-present`, then refuses the snapshot unless numbered UFW contains the exact comment `netductor-ssh-bootstrap`.
- A factory or previously provisioned host that already allows `22/tcp` under another comment (`factory-ssh`, `OpenSSH`) fails closed: `active UFW is missing the baseline factory SSH transition rule`.
- Manual workaround used: `ufw allow 22/tcp comment netductor-ssh-bootstrap`.
- Fix later: treat any existing `22/tcp ALLOW` as the transition rule, or insert the tagged rule ourselves when `:22` is already allowed. Do not require the operator to rename a comment.

Finalize (same file, `set -e`):

- `ufw allow` / `ufw insert` are not idempotent. On an already-provisioned host they print `Skipping adding existing rule` / `Skipping inserting existing rule` and exit 1.
- Observed on retry after a partial run: allow skipped, insert skipped, bootstrap rule deleted (`Rule deleted`), then the script exited 1. Desired end state (52222 allow, 22 denied first) was not reached.
- Fix later: treat UFW "already exists" / skip as success (`|| true` only after confirming the rule is present), and make the deny-then-delete sequence safe to re-run. Verification must still fail closed if `:22` is not denied first after the sequence.

Same host, next stage (join host baseline, still no code change):

- After SSH had already moved to `:52222` key-only, the leftover `22/tcp ALLOW` comment `netductor-ssh-bootstrap` (added only to satisfy prepare) failed host-baseline preflight.
- Preflight requires the first WAN action on `22` to be `DENY` for IPv4 and IPv6, and no effective `tcp ALLOW from Anywhere` before that deny (`internal/firewall/firewall.go` `checkWAN("22", "DENY", "ssh22")`).
- So prepare and baseline disagree: prepare refuses without a tagged `22 ALLOW`, baseline refuses while that `ALLOW` is still first. A re-deploy of an already-hardened host cannot satisfy both.
- Fix later together with the finalize idempotency item: once key-only `:52222` is proven, prepare must not demand a factory `22 ALLOW`, and baseline must not run against a mid-transition policy.
