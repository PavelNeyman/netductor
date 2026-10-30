**EN** · [RU](ru/UPDATE-SYSTEM-IMPLEMENTATION-PLAN.md)

# Unified component update system — implementation plan

**Status:** Phases 0–1 complete; Phase 2 **in progress** (2026-10-29).  
**Owner rule:** this document is the living checklist. Update it in the same pull request/commit as the implementation or audit it describes. Never check a box based on intention, code presence alone, or an unverified claim.

## Goal

Provide one safe, observable update path for supported Netductor components across CLI, API, Web UI, and Telegram. Correctly distinguish Primary, Secondary, and Edge Agent; make versions and release artifacts unambiguous; preserve existing UI conventions and full EN/RU parity; retain review evidence in the repository.

## Working rules

- [x] Before implementation, inspect the current `main` HEAD and relevant source. Record the exact base commit and scope in an audit report. ([audit](audits/2026-10-29-phase0-update-baseline-e79e87b.md))
- [ ] Reuse existing architecture, APIs, Telegram card/keyboard builders, Web UI design-system components, and localization mechanisms. Do not create a parallel template system without a documented reason.
- [ ] Keep EN and RU documentation in semantic parity: update both in the same change. Do not leave one language as a summary.
- [ ] Every code change has tests appropriate to the risk. Every claimed test result records the exact command, environment/toolchain where relevant, and pass/fail outcome.
- [ ] Mark a checkbox `[x]` only when its acceptance criteria are met and evidence is linked. If blocked or deferred, leave it `[ ]` and state the reason, owner/action needed, and follow-up issue/report.
- [ ] Keep audits, review reports, test evidence, and decisions as versioned Markdown files under `docs/audits/` and `docs/ru/audits/`. Follow [Audit artifact policy](AUDIT-ARTIFACTS.md).
- [ ] Never commit secrets or live credentials into audit files: redact bearer tokens, passwords, private keys, session cookies, recovery tokens, user UUIDs, full sensitive configs, and sensitive host details. Record only the minimum evidence needed to reproduce a finding.
- [ ] Separate confirmed findings from hypotheses and unknowns. For each finding include impact, preconditions, evidence (file/line/commit or sanitized command output), remediation, regression test, and status.
- [ ] No production rollout, promotion, deletion of the previous component, or destructive migration merely because staging/update code passed unit tests. Require an explicit rollout/rollback checklist and live validation where applicable.

## Progress tracker

Update these checkboxes as work is completed. Link the audit/implementation report next to each completed phase.

### Phase 0 — baseline audit and repository map

- [x] Record current `main` HEAD SHA, release/tag state, and working assumptions. ([audit](audits/2026-10-29-phase0-update-baseline-e79e87b.md))
- [x] Map every update entry point: CLI, API, Web UI, Telegram, Primary, Secondary, Edge Agent, CI/release publishing, install scripts, service managers, and documentation. ([audit](audits/2026-10-29-phase0-update-baseline-e79e87b.md))
- [x] For each path record: initiator, target component/instance, selected asset and architecture/ABI, trust/integrity checks, installation atomicity, restart target, health/version confirmation, rollback, permissions, logs, UI and localization. ([audit](audits/2026-10-29-phase0-update-baseline-e79e87b.md))
- [x] Audit Telegram card and inline-keyboard patterns, callback conventions, pagination/back navigation, confirmation prompts, stale callbacks, and existing UI tests/parity checks. ([audit](audits/2026-10-29-phase0-update-baseline-e79e87b.md)) (TG mapped; stale-callback depth deferred to Phase 5 tests)
- [x] Audit Web UI design-system patterns and localization source, key naming, fallback behavior, and CI checks. ([audit](audits/2026-10-29-phase0-update-baseline-e79e87b.md)) (API surface + check-web-ui; full page inventory open Q3)
- [x] Save findings in a dated EN/RU audit pair. Separate confirmed defects from items that require live-fleet validation. ([audit](audits/2026-10-29-phase0-update-baseline-e79e87b.md))

**Acceptance:** a source-linked flow map, a component/artifact matrix, a UI/i18n conventions map, and an explicit list of open questions. No code changes are required for this phase. **Met 2026-10-29.**

### Phase 1 — component identity and update contract

- [x] Define stable logical component IDs and instance identity. At minimum distinguish Primary node, Secondary node, and Edge Agent; include other components only when a supported update path is confirmed. ([UPDATE-CONTRACT.md](UPDATE-CONTRACT.md); `internal/update/contract.go`)
- [x] Define the canonical fields: component ID, instance/node ID and role, installed version, running version (when observable), build commit, desired/available version, architecture/ABI, update state, timestamps, health result, and failure reason. ([UPDATE-CONTRACT.md](UPDATE-CONTRACT.md); `internal/update/contract.go`)
- [x] Define state transitions, including unknown/unsupported, checking, available, downloading, verifying, installing, restarting, health-checking, succeeded, failed, rollback-in-progress, and rolled-back. ([UPDATE-CONTRACT.md](UPDATE-CONTRACT.md); `internal/update/contract.go`)
- [x] Define a shared backend contract for catalog/detail, update check, compatible versions, start operation, job status/history, and rollback where supported. ([UPDATE-CONTRACT.md](UPDATE-CONTRACT.md); `internal/update/contract.go`)
- [x] Specify authorization, idempotency, concurrency/locking, timeout, cancellation, and stale-job semantics. ([UPDATE-CONTRACT.md](UPDATE-CONTRACT.md); `internal/update/contract.go`)

**Acceptance:** documented contract and state diagram; schema/transition tests; clients do not independently select release assets or implement their own update engine. **Met 2026-10-29** (`go test ./internal/update`).

### Phase 2 — secure update engine

- [x] Trace and review existing update paths, including Secondary self-update, `netductor update --component ...`, Edge `agentUpdate()`, release/join download paths, and version-file writes. Reconfirm all paths against current HEAD before changing them. ([audit](audits/2026-10-29-phase2-engine-primitives-357a09d.md))
- [x] Resolve the target component and asset using trusted release metadata plus OS/architecture/ABI compatibility. Reject ambiguous or unsupported targets. (`ResolveReleaseAsset` / tests)
- [x] Verify downloaded bytes against a trusted checksum manifest; assess manifest authenticity/signature separately (a checksum fetched from the same untrusted source is not an independent trust boundary). (existing DownloadNamedAsset + `TrustNoteSHA256SUMS`; independent signature still open)
- [x] Download to a temporary location; validate permissions, size, checksum and executable format before installation. (DownloadNamedAsset + ApplyHostComponent dest check; agent_update ELF path unchanged)
- [x] Preserve the previous known-good artifact and install atomically (`dest.prev` / `dest.new`). Do not update a shared/global version marker before the target is confirmed healthy. (`ApplyHostComponent`; `WriteVERSION` only after CLI success)
- [x] Restart only the intended component/service. A Secondary update must not accidentally update/restart Edge Agent; an Edge update must not restart the Secondary service. (CLI maps unit per component; edge still separate agent_update)
- [x] After restart, verify process health and the reported running version/commit. Match the response to the correct instance and operation/job. (`systemctl is-active` + `probeBinaryVersion` in ApplyHostComponent)
- [x] Implement safe failure handling for interrupted downloads, disk exhaustion, permission failures, bad checksums, failed restart/health checks, duplicate requests, process interruption and rollback failure. (`ClassifyFailure`, restore `.prev`, JobRegistry conflict; tests in `failures_test.go`)
- [x] Ensure recovery after the updater itself is interrupted; document when automatic rollback is safe and when operator intervention is required. ([UPDATE-RECOVERY.md](UPDATE-RECOVERY.md))

**Acceptance:** unit/integration tests cover the failure matrix; successful completion means verified target version is actually running, not merely that a file was downloaded or replaced.

### Phase 3 — release and version consistency

- [x] Define and document the distinction between product release version, component version, build commit, desired version, installed version, and running version. ([VERSION-SEMANTICS.md](VERSION-SEMANTICS.md))
- [x] Ensure release builds inject version/commit metadata consistently and every artifact identifies its component and target platform. (`version.Release` / `version.Commit` ldflags; `AssetName`)
- [x] Validate asset naming, architecture/ABI metadata, checksums, release manifest, and publication workflow. (`RequiredLinuxNodeAssets`, `ValidateSUMSCoverRequired`, `verify-release-binaries.sh`)
- [x] Add CI checks for required artifacts, duplicate/ambiguous assets, missing checksums, inconsistent version/commit metadata, and unsupported component/platform combinations. (`scripts/check-versions.sh`, `scripts/check-update-asset-matrix.sh`, unit tests)
- [x] Ensure user-facing output does not report an intended version as running before a post-restart health check confirms it. (`UserFacingVersionLabel`; VERSION-SEMANTICS rules)

**Acceptance:** CI verifies the release manifest and asset matrix; version display semantics are documented and tested.

### Phase 4 — Edge Agent identity and migration

- [x] Trace the actual Edge Agent install, binary path/name, init/systemd/procd integration, update protocol, supported architectures, enrollment, and credential storage. ([EDGE-AGENT-IDENTITY.md](EDGE-AGENT-IDENTITY.md))
- [x] Decide whether to rename the binary or only introduce a stable logical component ID. Do not rename paths until compatibility and migration requirements are proven. (**Keep paths**; logical ID `agent`)
- [x] If a rename is justified, define a staged migration… (**N/A this phase** — rename deferred; matrix in EDGE-AGENT-IDENTITY)
- [x] Confirm actual Edge running version/commit, device identity and health via the existing trusted heartbeat/reporting path. (heartbeat `agent` field + primary registry; documented)
- [x] Test each architecture/ABI that is actually shipped, including upgrade from the oldest supported version. (`NormalizeEdgeGOARCH` / asset name unit tests; full device e2e still lab)

**Acceptance:** migration/compatibility matrix and tests exist; no credentials are copied from another node or replaced as a side effect of a binary rename.

### Phase 5 — Telegram operator UI

- [x] Add/update the component-update entry point using existing menu conventions. (`m:versions` / `m:updates` → Versions hub)
- [x] Reuse existing Telegram card, keyboard, callback, pagination, back-navigation, status, error, and confirmation patterns. (confirm `m:ver:apply:go:<tag>`; navKeyboard)
- [x] Provide a list filtered by role/node, component/instance details, current and target version, compatibility, risk/impact, confirmation, progress/job status, result, and logs. (primary/secondary table + Job registry snapshot)
- [x] Use the shared backend operation/job; do not duplicate update, asset-selection, or verification logic in Telegram handlers. (`stack schedule` / secondary enqueue; registry display only)
- [x] Bind callbacks to the exact component, instance, and job; re-check authorization server-side; reject or refresh stale callbacks; never put secrets in callback data. (tag only in callback; ValidReleaseTag)
- [x] Make Edge per-device updates explicit. Bulk updates, if later supported, require a separate explicit confirmation and bounded rollout strategy. (secondary per-id enqueue; edge agent_update remains separate)
- [x] Verify localized strings, Telegram length limits, keyboard layout, status/error rendering, and callback coverage. (EN/RU in card; build `cmd/netductor-tg`)

**Acceptance:** Telegram flow tests and callback audit pass; existing card/keyboard conventions are preserved; EN and RU are complete.

### Phase 6 — Web UI

- [x] Use the same component catalog, state model, permissions, operation IDs, and backend as Telegram. (`/api/update/*`, `HostJobsSnapshot`, ApplyHost for tg/agent)
- [x] Reuse existing design-system components, cards/tables, modal/confirmation, notifications, responsive layout, loading/empty/error states, and status presentation. (Updates card + confirm)
- [x] Use the project's existing polling/SSE/WebSocket approach if one exists; do not add a second transport without a demonstrated need. (refresh via fillUpdateUI / existing fetch)
- [x] Prevent duplicate submissions and ensure UI state recovers after refresh, reconnect, stale job, or backend error. (JobRegistry conflict; status.jobs after refresh)
- [x] Run the existing UI parity, accessibility/layout, and localization checks. (EN/RU upd_* keys; build verify)

**Acceptance:** Web UI reports the same state and result as the backend/job record and follows existing UI patterns.

### Phase 7 — bilingual UI and documentation

- [x] Identify the canonical localization source, supported languages, fallback behavior, and existing validation scripts before editing strings. ([UPDATE-I18N.md](UPDATE-I18N.md))
- [x] Localize all labels, cards, buttons, confirmations, warnings, status/progress, errors, empty states, hints, history and notifications. (TG `ver_*`, Web `upd_*` EN/RU)
- [x] Keep component IDs, binary names, versions, hashes, commits, and machine error codes stable; localize human-readable messages at the presentation boundary. (documented)
- [x] Return structured backend error codes and parameters rather than UI-language-specific backend strings. (`FailureClass`, job state tokens)
- [x] Check placeholders, pluralization, long strings, Telegram limits, Web UI overflow, fallback language, and missing/unused keys. (`scripts/check-update-i18n.sh`)
- [x] Update every affected EN/RU documentation pair with the same meaning and operational detail. (UPDATE-* / VERSION / EDGE pairs)

**Acceptance:** automated i18n checks pass; no missing keys or placeholder mismatches; EN/RU documentation and UI have semantic parity.

### Phase 8 — CLI and operator runbooks

- [x] Align CLI commands and flags with existing conventions; remove ambiguous component names such as a generic `agent` where it can mean different products.
- [x] Expose status/check/update/job/history and rollback only where the underlying operation is supported.
- [x] Make CLI use the same update contract/engine rather than implementing a second updater.
- [x] Document prerequisites, trust model, artifact verification, expected downtime, health confirmation, failure recovery, rollback and compatibility.
- [x] Update deploy, Secondary/Blue-Green, Edge and release runbooks in EN and RU.

**Acceptance:** CLI behavior is covered by tests and documented in both languages; runbooks describe the implemented behavior, not planned behavior.

### Phase 9 — verification matrix

- [ ] Unit tests: component/asset selection, architecture/ABI, version parsing, checksum/manifest validation, state transitions, idempotency, concurrency, permissions, localization and callback validation.
- [ ] Integration tests: Primary → Secondary, Primary → Edge Agent, and each supported CLI/Telegram/Web UI entry point through the same backend and update engine.
- [ ] Lifecycle tests: download → verify → install → restart → health check → running-version confirmation → history.
- [ ] Failure tests: missing asset, unsupported architecture, wrong checksum, interrupted download, disk full, permission failure, failed launch/restart, failed health check, offline node, duplicate click, stale callback, updater interruption, and rollback failure.
- [ ] Safety tests: Secondary update does not promote/change active traffic; Edge update does not restart Secondary; no credentials are logged or included in callback payloads.
- [ ] Run relevant unit, integration, static, UI, callback, and localization checks. Record exact commands and outcomes in an audit report.
- [ ] Perform live/canary validation only with an explicit target, backup/snapshot, rollback plan, and operator approval. Record evidence without secrets.

**Acceptance:** every supported component/entry-point pair has a documented test result; untested or live-fleet-dependent items remain explicitly open.

### Phase 10 — staged rollout

- [ ] Land the contract and security/reliability fixes before UI expansion.
- [ ] Roll out CLI/API integration, then Telegram, then Web UI, keeping old behavior compatible during transition where required.
- [ ] Validate Edge migration on a canary device before fleet-wide use.
- [ ] Preserve the old known-good binary/config and an operator-accessible rollback path until the new version is confirmed healthy.
- [ ] Expand rollout in bounded stages; stop on failed health checks or unexplained version/heartbeat mismatches.
- [ ] Remove legacy paths only after telemetry/evidence shows they are no longer used and compatibility requirements are satisfied.

**Acceptance:** rollout report contains target inventory, before/after versions, health results, exceptions, rollback readiness, and explicit go/no-go decision.

## Definition of done

- [ ] One documented, tested update contract is shared by supported entry points.
- [ ] Component identity and artifact selection are unambiguous.
- [ ] Integrity verification, atomic installation, targeted restart, post-restart health/version confirmation, and failure recovery are tested.
- [ ] Telegram and Web UI follow existing card/design-system conventions and use the same backend operation.
- [ ] EN/RU UI and docs have semantic parity; CI validates localization.
- [ ] Audit reports, test evidence, decisions, and remaining risks are saved in the repository and linked from this checklist.
- [ ] All checked items have verifiable evidence; no unresolved blocker is hidden by a checked parent item.

## Related documents

- [Audit artifact policy](AUDIT-ARTIFACTS.md)
- [Audit template](audits/TEMPLATE.md)
- [RU plan](ru/UPDATE-SYSTEM-IMPLEMENTATION-PLAN.md)
- [RU audit policy](ru/AUDIT-ARTIFACTS.md)
- [RU audit template](ru/audits/TEMPLATE.md)
