# Plan: internal releases (bare git + CI + registry) + op source build

**Status:** P0 done (0.9.279); P1 build-local in progress  
**Context:** GitHub remains the development remote (agent access). Day-2 artifacts should not depend on uploading every tag to GitHub Releases. Primary already has bare git, isolated CI (docker), and local OCI registry (`127.0.0.1:5000`).

## Goals

1. Build release-shaped binaries **on primary** from a git tag/sha without installing a host Go toolchain.
2. Install via the same paths as today (`stack apply`, `EnsureAgentBinary`) preferring **local artifacts**.
3. Optionally build on **operator machine** (`netductor-op`) from the same sources for offline / Mac op binary.
4. Keep GitHub as source-of-truth for development and optional fallback download.

## Non-goals

- Dropping GitHub for development.
- Building agent arches on the OpenWrt device.
- Requiring public GH Release for every internal build.

## Existing pieces

| Piece | Location |
|--|--|
| Bare git | `/var/lib/netductor/git` (doctor: `git root`) |
| CI isolate | `internal/ci` — `golang:*` container, `NETDUCTOR_CI_HOST` discouraged |
| Registry | `internal/registry` — `netductor-registry` on `127.0.0.1:5000` |
| Local build scripts | `scripts/release.sh`, `scripts/build-release-local.sh` |
| Download / apply | `internal/download`, `internal/stack`, `EnsureAgentBinary` |

## Target flow

```text
GitHub (dev) ──fetch/mirror──► bare git on primary
                                    │
                                    ▼
                         CI job (docker golang)
                         multi-arch matrix ≈ release.sh
                                    │
              ┌─────────────────────┼─────────────────────┐
              ▼                     ▼                     ▼
   /var/lib/netductor/     registry 127.0.0.1:5000    (optional) GH Release
   releases/<tag>/         netductor/…
   + SHA256SUMS
              │
              ▼
    stack apply / update / EnsureAgentBinary
    order: local store → registry → GitHub
```

## Phases

### P0 — Local artifact store (no builder yet)

- Dir layout: `/var/lib/netductor/releases/<tag>/` same asset names as GH + `SHA256SUMS`.
- `download` / stack / agent ensure: **try local first**, then GitHub.
- CLI: `netductor release list-local`, `release import <dir|tarball|url>`.
- Manual path: build on Mac with `scripts/release.sh`, `release import` on primary.

### P1 — Build tag on primary (docker CI)

- Job: checkout tag/sha from bare git → `ci.Isolate` with `ImageForGo()` → build matrix (node, tg, agents multi-arch, op linux-amd64).
- Write outputs under `releases/<tag>/` + `SHA256SUMS`.
- Optional: `crane push` images to local registry (later; binaries first).
- Trigger: `netductor release build <tag|sha>` + API; log under `/var/lib/netductor/ci/`.

### P2 — Mirror sources

- Periodic `git fetch` GitHub → bare `netductor.git` (deploy key or token on primary).
- Tag list for UI/CLI from **local** git; fallback GitHub API if mirror stale.

### P3 — Operator source build

- `netductor-op release build --tag vX.Y.Z`:
  - clone tag (GH or configurable git URL);
  - local `go build` for needed GOOS/GOARCH;
  - install into `~/.cache/netductor/agents` / offline pack;
  - optional: push/import to primary local store.
- Darwin `netductor-op` from source without GH asset.

### P4 — Thin UI

- Tools / op / API: list tags, start build, show job status, apply tag.
- No business logic in UI — backend only.

## Implementation notes

- Reuse asset naming and SHA256SUMS verification from `internal/download` / update.
- CI must stay isolated (`internal/ci`); never default `NETDUCTOR_CI_HOST=1`.
- Version pin: built binaries should embed the same tag in `internal/version.Release` for that build.
- mipsle/arm agent: cross-compile from amd64 primary container (same as `release.sh`).

## Order of work

1. P0 local store + prefer-local (immediate value).
2. P1 `release build` via docker.
3. P3 op build.
4. P2 mirror + P4 UI.

## Acceptance

- [ ] `stack apply vX` works with **only** local `releases/vX` present (no GH).
- [ ] `release build vX` on primary produces SHA256SUMS-matching matrix in docker.
- [ ] Op can produce agent binary for offline edge without GH Release upload.
- [ ] GitHub Releases remain optional fallback.


## Distribution: how secondaries / op get builds

| Channel | How | Pros | Risks |
|--|--|--|--|
| **A. GitHub Releases (optional)** | `release.sh` upload as today | Works from anywhere; agent/CI already use it | Public binaries; token for private repo; rate limits |
| **B. Local store only + VPN** | Artifacts on primary under `releases/`; secondary/op pull via **agent mTLS :8789** or existing stack path over tunnel | No new public port; same trust as agent plane | Secondary must reach primary control plane; pure offline edge needs op cache |
| **C. Public HTTPS hole** | nginx/caddy `:443/path` → releases | Simple curl | **Avoid**: expands attack surface, needs auth or secret URLs, leaks versioning |

**Decision (default):** **B first**, **A optional mirror**.

- Primary `stack apply` / `update apply`: read `/var/lib/netductor/releases/<tag>/` before GitHub.
- Secondary agent upgrade command: primary **pushes** or agent **pulls through existing mTLS API** (same as today agent_update), not a new open port.
- Op offline: build/import on Mac → `~/.cache/netductor/agents` or `scp` into primary `release import`.
- Do **not** expose registry `5000` or releases dir on WAN.

### Automation later

1. Hook: tag on bare git → CI build job → write `releases/<tag>` → `notify.AlertOnce("release:built:"+tag, …)`.
2. Doctor/metrics: `local_release_latest`, age of mirror, last build status.
3. TG Tools → Releases: list-local, apply (calls stack with local prefer).
4. Optional: after successful local build, **optional** GH upload (flag), not required for prod.

### Prod rollout of this feature

1. Ship node with local-prefer (`DownloadNamedAsset`).
2. Import current GH release once: `release import vX ./dist`.
3. Alerts already cover stack apply / agent update failures — reuse, add `release:import` / `release:build` keys.
4. Monitor: disk under `releases/` (retention: keep last N tags), CI job fail → TG.

### Security notes

- SHA256SUMS required for local same as GH (unless explicit skip).
- Import only from operator path (CLI/API session), not anonymous.
- Registry remains loopback; if auth later, htpasswd already supported in `internal/registry`.


### P1 status (implemented in tree)

- `netductor release build <tag> [--skip-darwin]` → `internal/update.BuildLocal`
- Isolated via `ci.Exec` + `ImageForGo()`; artifacts → `ImportReleaseDir`
- TG alerts: `release:built:*` / `release:build-fail:*`
- Source: shallow `git clone --branch <tag>` (GitHub); bare-git checkout can replace later (P2)
