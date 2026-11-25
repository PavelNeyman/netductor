# Plan: Operator job log stream (deploy → Web / TUI)

**Status:** P0 implemented (0.9.773); P1–P2 open  
**Created:** 2026-11-25  
**Related:** [PLAN-THIN-UI-JOBS.md](PLAN-THIN-UI-JOBS.md), edge provision, `internal/deploy`, `internal/jobs`

## Problem

Web / TUI job panels show **steps** (`setStep`) and whatever is passed to **`appendLog`**. Almost all deploy progress (`==> storage:…`, SSH remote lines, FORMAT/EXTROOT markers) is written only to **`os.Stderr`** of the `netductor-op` process.

Observed on edge provision:

| Source | Visible in Web job log |
|--------|-------------------------|
| `setStep(provision, "starting…")` | yes |
| `fmt.Fprintln(os.Stderr, "==> …")` in `internal/deploy` | **no** |
| `runSSHOnPort` / `runSSHOnPortLogged` → stderr | **no** |
| probe/bootstrap `appendLog(raw)` after SSH returns | yes (batch, not live) |

Consequence: UI shows a single frozen step (e.g. `configure extroot`) while useful output is only on the Mac terminal running `netductor-op serve`. Cancel works; diagnosis does not.

v0.9.772 fixed **silent hang** (removed `apk add` from extroot path, streamed SSH to process stderr). That does **not** fill the Web log.

## Goal

Live (or near-live) deploy/SSH lines in **OperatorJobStore** log so Web and TUI poll the same text an operator would see on stderr — without requiring a separate terminal.

Non-goals:

- Full interactive PTY / xterm (separate backlog item in OPEN_ITEMS)
- Changing primary VPS job API shape
- Logging secrets (passwords, keys) into job log

## Design

### Principle

Do **not** hijack global `os.Stderr` for the whole process (races with serve, other jobs).

Inject an optional logger into deploy:

```text
type LogFunc func(line string)

// EdgeOpts / Deploy options
Log LogFunc  // nil → Fprintln(os.Stderr) only (CLI behavior unchanged)
```

Helpers:

```go
func deployLog(o EdgeOpts, format string, args ...any) {
  line := fmt.Sprintf(format, args...)
  if o.Log != nil {
    o.Log(line)
  }
  fmt.Fprintln(os.Stderr, line) // keep CLI / serve console
}
```

SSH path (`runSSHOnPortLogged`): accept `io.Writer` (or LogFunc). Stream remote lines to that writer **and** stderr.

### Wiring (operator jobs)

`runEdgeProvision` (and later bootstrap/probe live):

```go
log := func(line string) { appendLog(strings.TrimRight(line, "\n")) }
// pass into DeployEdge via EdgeSpec / EdgeOpts
```

`appendLog` already persists into the job and is what `GET /v1/jobs/{id}` exposes as log.

### Surfaces

| Phase | Scope |
|-------|--------|
| **P0** | `edge.provision` → `DeployEdge` + storage + SSH logged path |
| **P1** | `edge.bootstrap` live lines (not only final `appendLog(raw)`) |
| **P1** | `edge.probe` optional live (today batch raw is enough) |
| **P2** | primary/secondary/buildhost deploy same pattern if still stderr-only |
| **P2** | optional SSE `GET /v1/jobs/{id}/log` if poll lag bothers (API already sketched in thin-UI plan) |

### Safety

- Do not append router passwords, SSH private material, or full mTLS PEMs
- Cap line rate / max log bytes per job if needed (existing job store limits if any)
- Strip OpenSSH PQ noise already handled for error paths; keep strip for stored log if noisy

### Acceptance

1. Edge deploy with overlay_ext: Web job log shows `EXTROOT_BEGIN` → `COPY_*` → `EXTROOT_OK` **while** the job is running (not only on completion).
2. CLI `netductor-op` without jobs still prints the same lines on stderr.
3. Cancel still stops SSH via existing `WithSSHContext`.
4. No regression on probe/bootstrap success paths.

## Implementation sketch

1. Add `Log func(string)` to `deploy.EdgeOpts` (and thin wrapper from `operator.EdgeSpec`).
2. Replace critical `Fprintln(os.Stderr, …)` in `edge_run.go` / `edge_storage.go` with `deployLog`.
3. `runSSHOnPortLogged(..., w io.Writer)` — MultiWriter(stderr, w).
4. `runEdgeProvision`: bind `appendLog` → opts.Log.
5. Manual lab: Cudy provision, watch Web log update every poll interval.
6. Optional unit test: Log callback receives a fixed marker from a dry-run path.

## Out of scope / follow-ups

- Web terminal (xterm) — OPEN_ITEMS “Web terminal”
- Primary job log parity on VPS API
- Restructuring thin-UI poll interval

## Refs

- Discussion 2026-11-25 (Cudy extroot hang diagnosis + job log gap)
- Release **v0.9.772** — storage hang fix only; log stream still open
