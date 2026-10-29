# Version semantics (Phase 3)

| Term | Meaning | Source |
|------|---------|--------|
| **Product release** | Marketing/tag version for a GitHub Release (`v0.9.x`) | `VERSION`, `internal/version.Release`, git tag |
| **Component version** | Same product train for node/tg/agent/op in this project (single train) | Asset built with same `-X …Release=` |
| **Build commit** | Git SHA baked at link time when provided | `version.Commit` (`-X …Commit=`) |
| **Desired version** | Target of an update job | `UpdateJob.TargetVersion` / CLI tag |
| **Installed version** | What is on disk after a successful replace | Binary file; `/etc/netductor/VERSION` after CLI writes it |
| **Running version** | What the live process reports after restart + health | `systemctl` + `probeBinaryVersion` / `version.Running()` |

## Rules

1. **Do not** show Desired as Running until health (and version probe when enabled) succeeds.
2. `WriteVERSION` runs only after apply success (CLI), not mid-download.
3. `version.Running()` may prefer `/etc/netductor/VERSION` over an older binary probe — still not proof until health after *this* apply.
4. Clients never invent asset names — only `AssetName(component, platform)`.

## UI copy

- While job state is `downloading`…`health_checking`: show **target** / **in progress**, not “updated to X”.
- On `succeeded`: show **running X**.
- On `failed` / `rolled_back`: show previous / error class, not target as current.
