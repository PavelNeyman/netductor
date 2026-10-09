

# CI workflows (netductor VPS)

## Resolve path (locked)

| Rule | |
|--|--|
| Directory | **`.github/workflows/`** only |
| File + `name:` | **= project name = GitHub repo name** (`larvatus.yml`, `name: larvatus`) |
| Resolve | exact `.github/workflows/<Name>.yml` or `.yaml` only — **no** `ci/`, **no** first-alphabetical |
| `ci/` folder | **removed** — not resolved |
| `git project add` | Name = repo name; default workflow path set automatically |
| host | `vps` (default) builds on primary; `mac` → **queue + TG/CLI**, no compile on VPS |
| Windows | `GOOS=windows GOARCH=amd64` inside linux container job (Go); no MSI/WiX on VPS |

Mac queue CLI:

```bash
netductor git project build niimbot-ios v1.2.3   # enqueues if host=mac
netductor git project queue list
netductor git project queue done <id>
```

Operator Mac (from TG notification):

```bash
netductor-op project build niimbot-ios --ref v1.2.3
# then import artifacts to VPS (release import / scp)
```

---

# Netductor CI workflows (GHA-compatible subset)

**Audience:** humans and other AI agents adding CI to a repo that will be built on a netductor primary (`netductor git project build` / post-receive / `git workflow`).

**Runner:** primary VPS, isolated in **Docker/Podman** by default (`internal/ci` + `internal/gha`). Host toolchains are **not** required.

This is **not** full GitHub Actions. Only the features below are executed.

---

## How netductor picks a workflow

For a registered **project** (`netductor git project add …`):

1. Explicit `workflow` path on the project (e.g. `.github/workflows/ci.yml`)
2. Else first file under `.github/workflows/*.{yml,yaml}` (sorted by name)
3. Else shell **pipeline** name (`--pipeline go-test` → `/etc/netductor/git-pipelines/…`)
4. Else error / empty

**Recommendation for third-party repos:** put one file at:

```text
.github/workflows/ci.yml
```

Use a name that sorts first if you keep several files (`ci.yml` before `release.yml`).

Register on primary:

```bash
netductor git project add myapp org/myapp
# or pin workflow:
netductor git project add myapp org/myapp --workflow .github/workflows/ci.yml
netductor git project sync myapp
netductor git project build myapp
```

Upstream may be `org/repo` or full `https://github.com/org/repo.git`.

---

## Supported YAML shape

```yaml
name: optional-title

# `on:` is accepted and IGNORED (no push/PR triggers inside netductor yet)
on: [push]

jobs:
  # Job names are run in sorted alphabetical order
  build:
    runs-on: ubuntu-latest   # accepted, ignored (always container/host policy)
    # Container is THE way to choose language toolchain
    container: golang:1.22-bookworm
    # or:
    # container:
    #   image: node:20-bookworm
    env:                      # job-level env → all steps
      CGO_ENABLED: "0"
    steps:
      - name: optional label
        run: |
          set -euo pipefail
          go test ./...
        # shell: bash          # default bash
        # working-directory: backend
        # env:
        #   FOO: bar

      # uses: actions/checkout@v4   # SKIPPED (checkout is already done by netductor)
      # uses: actions/setup-node@v4 # SKIPPED — use container: instead
```

### Supported fields

| Field | Support |
|--|--|
| `name` (workflow) | yes (logging) |
| `on` | parsed, **ignored** |
| `jobs.<id>` | yes; order = **sorted job id** |
| `jobs.*.runs-on` | ignored |
| `jobs.*.container` | string **or** `{ image: "…" }` |
| `jobs.*.env` | yes |
| `jobs.*.steps[].name` | yes |
| `jobs.*.steps[].run` | **required** to execute (multi-line OK) |
| `jobs.*.steps[].shell` | yes (default `bash`) |
| `jobs.*.steps[].working-directory` | relative to repo root |
| `jobs.*.steps[].env` | yes |
| `jobs.*.steps[].uses` | **skipped** (log only) |
| `jobs.*.steps[].id` | ignored |
| matrix / needs / services / artifacts upload | **no** |
| `actions/checkout`, `setup-*` | **no** — checkout is external; use `container:` |

If `container` is omitted and isolation is on, image is **auto-detected** from worktree:

| Marker | Image (default) |
|--|--|
| `go.mod` | `golang:1.22-bookworm` |
| `package.json` | `node:20-bookworm` |
| `Cargo.toml` | `rust:1.81-bookworm` |
| `pyproject.toml` / `requirements.txt` | `python:3.12-bookworm` |
| else | `debian:bookworm-slim` |

Override globally: `NETDUCTOR_CI_IMAGE_GO`, `_NODE`, `_RUST`, `_PYTHON`, `NETDUCTOR_CI_IMAGE`.

Escape hatch (discouraged on prod): `NETDUCTOR_CI_HOST=1` runs `run:` on the host.

---

## Examples for other AI agents

### Go

```yaml
name: go-ci
on: [push]
jobs:
  test:
    runs-on: ubuntu-latest
    container: golang:1.22-bookworm
    steps:
      - name: test
        run: |
          set -euo pipefail
          go test ./...
      - name: build
        run: |
          set -euo pipefail
          CGO_ENABLED=0 go build -o /tmp/app ./cmd/app
```

### Node

```yaml
name: node-ci
jobs:
  test:
    container: node:20-bookworm
    steps:
      - name: ci
        run: |
          set -euo pipefail
          npm ci
          npm test
```

### Python

```yaml
name: py-ci
jobs:
  test:
    container: python:3.12-bookworm
    steps:
      - name: test
        run: |
          set -euo pipefail
          pip install -r requirements.txt
          pytest -q
```

### Multiple jobs (order = alphabetical job keys)

```yaml
name: multi
jobs:
  a_lint:
    container: golang:1.22-bookworm
    steps:
      - run: go vet ./...
  b_test:
    container: golang:1.22-bookworm
    steps:
      - run: go test ./...
```

(`a_lint` runs before `b_test` because of sorted names.)

### Do **not** rely on

```yaml
steps:
  - uses: actions/checkout@v4     # skipped
  - uses: actions/setup-node@v4   # skipped — use container: node:20
  - run: echo ${{ github.sha }}   # expressions not expanded
```

Checkout is already at the worktree root when steps run. `GITHUB_WORKSPACE`, `REPO_NAME`, `REPO_PATH`, `CI=true` may be set in the environment.

---

## Shell pipelines (fallback, not in-repo)

If the repo has **no** usable workflow, register a pipeline on the project:

```bash
netductor git project add legacy org/legacy --pipeline go-test
```

Managed samples live under `/etc/netductor/git-pipelines/` (`go-test`, `oci-push`, …). Prefer **in-repo** `.github/workflows/ci.yml` so the project is self-contained for any AI cloning the repo.

---

## Local / primary commands cheat sheet

```bash
netductor git project list
netductor git project add NAME UPSTREAM [--workflow PATH] [--pipeline NAME] [--build-on-fetch]
netductor git project sync NAME
netductor git project build NAME [ref]
netductor git workflow NAME [path]          # ad-hoc bare repo
netductor ci status
```

API (session): `GET/POST /api/git/projects`, `POST …/sync`, `POST …/build`.

Artifacts of workflow runs may appear under `/var/lib/netductor/git-artifacts/`.

---

## Guidance for coding agents

1. Always add `.github/workflows/ci.yml` with **only** `run:` steps and an explicit **`container:`**.
2. Never depend on `uses: actions/*` for setup or checkout.
3. Keep scripts `set -euo pipefail`; network in container uses docker bridge (can reach proxy.golang.org, npm, etc. unless primary blocks egress).
4. Do not assume GitHub-hosted runners, secrets from GitHub, or artifact upload actions.
5. One primary job is enough; split only if you need clear failure boundaries (remember job order is alphabetical).
6. For netductor’s own release matrix, prefer `netductor release build` / `build-host` — not this generic project CI.

