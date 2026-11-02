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
- [x] Firewall mutation lock acquisition now has a 30-second bounded variant, used by bootstrap and rollback so a competing firewall operation cannot block recovery indefinitely.
- [x] Added focused UFW rollback tests proving that only the transaction-created peer rule is removed and a pre-existing peer rule is retained.
- [ ] Focused tests have not been executed yet. The repository changes are source-reviewed only; no Actions workflow was dispatched.

## Upgrade compatibility audit — implementation update

- [x] Versioned the state-repair completion marker. Older installations contain the timeless `ok` marker; the new `v2` marker forces exactly one additional repair pass so files and invariants added to `RepairStateOnUpgrade` in later releases are not permanently skipped.
- [x] Added a regression test proving that an old `ok` marker triggers the new repair generation once, then becomes idempotent.
- [x] Peer registry loading now normalizes the original unversioned representation (version 0) to schema 1, but rejects unknown future schema versions. Saving also rejects unsupported versions instead of silently rewriting them.
- [x] Added tests for legacy unversioned peer registry compatibility and fail-closed behavior for schema 2.
- [ ] Remaining persisted formats and all version-negotiation call sites still require review. These changes have not been test-executed.
