**EN** · [RU](../ru/audits/2026-10-29-phase2-engine-primitives-357a09d.md)

# Phase 2 progress — path reconfirm + engine primitives

> Status: **in progress** (primitives + tests; not full failure-matrix acceptance)  
> Date: 2026-10-29  
> Base parent: `357a09d` (Phase 1)  
> Scope: reconfirm update paths; ResolveReleaseAsset; JobRegistry

## Path reconfirm (HEAD)

| Path | Location | Notes |
|------|----------|-------|
| CLI node/tg/agent | `cmd/netductor/cli_misc.go` `runUpdate` | Uses `internal/update` download + optional restart |
| API | `cmd/netductor/api_update.go` | Same engine |
| Edge agent_update | `cmd/netductor-agent/agent_ops.go` | URL\|sha\|confirm; ELF; `.prev`; rollback |
| Edge arg fill | `cmd/netductor/cli_edge.go` `buildAgentUpdateArg` | SHA256SUMS + arch |
| Op self-update | `cmd/netductor-op/cli_update.go` | Darwin / brew |
| Release publish | `scripts/release.sh` | Clean tree, tag HEAD, post SHA |
| SHA helpers | `internal/update/github.go` `ParseSHA256SUMS`, `DownloadNamedAsset` | Skip-verify escape hatch remains |

## Trust

`TrustNoteSHA256SUMS` in `resolve.go`: SUMS = channel integrity only, not independent provenance.

## New API

- `ResolveReleaseAsset` / `ResolveReleaseAssetLocal`
- `JobRegistry` Begin / Transition / Get

## Tests run

```text
go test ./internal/update/ -count=1
```

## Not done yet (Phase 2 open)

- Wire every apply path exclusively through Resolve + JobRegistry
- Full failure matrix integration tests (disk full, kill mid-install, …)
- Automatic health/version confirm after restart as hard gate for `succeeded`
