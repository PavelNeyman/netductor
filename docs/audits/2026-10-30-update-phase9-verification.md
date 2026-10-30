# Audit: Update system Phase 9 verification matrix

- **Date:** 2026-10-30
- **Base SHA:** 59a9b16afd9888aefca6cb0b70ccf5b2542df361
- **Scope:** unit/failure tests for `internal/update`; CLI/docs Phase 8; no live fleet

## Commands run

```bash
go test ./internal/update/ -count=1
bash scripts/check-versions.sh
```

## Unit coverage (this pass)

| Area | Evidence |
|------|----------|
| Component/asset selection | contract_test, resolve_test, TestSafetyAgentDoesNotUseNodeAssetName |
| Architecture | TestResolveUnsupportedArch, edge_identity NormalizeEdgeGOARCH |
| Version parsing | TestValidReleaseTag, TestUserFacingVersionLabel |
| Checksum/manifest | manifest_test, TestWrongChecksumLocal, TestClassifyMissingAssetAndChecksum |
| State transitions | contract_test transitions + rollback |
| Idempotency/concurrency | TestJobRegistryIdempotencyAndConflict, TestConcurrentBeginConflict |
| Lifecycle host apply | apply_host_test happy + atomic + probe |
| Failure matrix | failures_test + ClassifyFailure classes |
| Jobs snapshot CLI backend | TestHostJobsSnapshotEmptyAndAfterJob |

## Explicitly open (not closed by unit tests)

| Item | Reason |
|------|--------|
| Primary → Secondary e2e apply | Needs dual-VPS |
| Primary → Edge agent_update | Needs OpenWrt device |
| Live TG/Web click paths | Needs operator session |
| Disk-full / offline node | Environment-dependent |
| Secondary non-promote safety live | R5 / policy on stand |
| Canary rollout (Phase 10) | Operator approval |

## Secrets

No tokens or host keys in this report.

## Verdict

**Unit/failure matrix for host update engine: PASS (go test).**  
**Integration/live matrix: OPEN** — do not treat Phase 9 parent as fully done until stand evidence exists.
