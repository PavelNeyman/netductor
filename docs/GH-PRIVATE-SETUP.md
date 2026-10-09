**EN** · private GitHub + netductor tokens (operator runbook)

# Make GitHub private and wire netductor

Two different jobs need two different PATs (already supported):

| Secret file on primary | Purpose | Minimum rights |
|--|--|--|
| `/etc/netductor/secrets/github_token` | **Releases** — list/download netductor (and similar) **Releases** assets | see below |
| `/etc/netductor/secrets/github_token_repos` | **Repos** — mirror/clone private source, migrate-from-gh | Contents read (+ metadata) |

Fallback: if `github_token_repos` is empty, mirror uses `github_token`.

---

## 1. Decision: which repos go private

| Repo | Recommendation | If private, what breaks without token |
|--|--|--|
| **netductor** | Keep **public** *or* private + always local release store + token for operators | Agent/brew fallback to GH Releases fails without PAT or local store |
| **Other projects** (bots, apps, …) | **Private** is fine | Only mirror/sync/build on primary need `github_token_repos` |

**Safe default:** all non-netductor → private; netductor stays public **or** private with P0 local store preferred (already in product).

---

## 2. Make a repository private (GitHub UI)

For each repo under your user/org:

1. Open `https://github.com/<owner>/<repo>/settings`
2. Scroll to **Danger Zone** → **Change repository visibility** → **Private**
3. Confirm

Or bulk via org settings if you use an organization.

**After:** unauthenticated `git clone` / release download from the internet stops working. Primary must use PAT or already-mirrored bare git.

---

## 3. Create tokens (fine-grained preferred)

### A) Token **releases** (narrow)

GitHub → **Settings → Developer settings → Fine-grained personal access tokens → Generate**

- **Token name:** `netductor-releases`
- **Expiration:** your policy (90d / custom)
- **Resource owner:** your user (or org)
- **Repository access:** **Only select repositories** → select **netductor** (and any other repo whose **Releases** you download)
- **Permissions:**
  - Repository:
    - **Contents:** Read-only *(needed for some release APIs)*
    - **Metadata:** Read-only
  - *(If classic PAT instead: scope `repo` is wider than needed; or `public_repo` only if all stay public)*

Save the value once (`ghp_…` / `github_pat_…`).

### B) Token **repos** (source mirror)

- **Token name:** `netductor-repos`
- **Repository access:** all private repos you want on primary (or whole account)
- **Permissions:**
  - **Contents:** Read-only
  - **Metadata:** Read-only
  - Optional **Administration:** none
  - Do **not** grant delete/workflow write unless you intentionally push from VPS

Classic alternative: scope **`repo`** (full private repo access) — simpler, broader.

---

## 4. Install tokens on primary

```bash
# on primary (SSH)
install -d -m 700 /etc/netductor/secrets

# releases
netductor update github-token set-releases 'ghp_OR_github_pat_RELEASES'
# repos
netductor update github-token set-repos 'ghp_OR_github_pat_REPOS'

netductor update github-token status
# expect: releases=set repos=set (masked)
```

UI equivalents (session): Updates → GitHub token; Git → token fields (if present).

Files (mode **0600**):

- `/etc/netductor/secrets/github_token`
- `/etc/netductor/secrets/github_token_repos`

Env overrides (optional): `GITHUB_TOKEN`, `NETDUCTOR_GITHUB_TOKEN`.

---

## 5. Register / clone / pipeline / test build

### One project

```bash
# name local; upstream owner/repo OR full https URL
netductor git project add mybot PavelNeyman/mybot \
  --workflow .github/workflows/ci.yml
# or --pipeline shell-name if not using GHA-like yaml

netductor git project sync mybot          # mirror-fetch with repos token
netductor git project build mybot         # container CI on primary
netductor git project list
```

### Bulk from GitHub user

```bash
netductor git project migrate-from-gh PavelNeyman --dry-run
netductor git project migrate-from-gh PavelNeyman --sync
```

Skips `netductor` by default. Does not enable `build_on_fetch` unless you set it.

### Workflow YAML in each repo

See [CI-WORKFLOWS.md](CI-WORKFLOWS.md). Minimal Go example:

```yaml
name: ci
on: [push]
jobs:
  build:
    runs-on: ubuntu-latest
    container: golang:1.22
    steps:
      - run: go test ./...
      - run: go build -o dist/app .
```

Non-Go: `container: node:20` / `python:3.12` and your build commands.

### Verify private access

```bash
# should succeed with repos token after private
netductor git project sync mybot

# without token → auth error on private upstream
```

### netductor itself private

1. Prefer **local release store** (already): build/mirror on primary, agents pull local first.  
2. Keep releases token on primary for `stack apply` fallback to GH.  
3. Homebrew: first install still needs a reachable asset URL — either public release, or document `netductor-op` install from primary over VPN/SSH; do not rely on anonymous GH.

---

## 6. Checklist after flipping private

- [ ] `github_token` (releases) installed, status OK  
- [ ] `github_token_repos` installed  
- [ ] `git project list` shows projects  
- [ ] `git project sync <name>` OK for each private upstream  
- [ ] `git project build <name>` OK for at least one Go repo  
- [ ] `stack apply` still finds SHA256SUMS (local or GH with releases token)  
- [ ] Mac: brew/op update path understood if netductor is private  

---

## 7. Security notes

- Tokens never in git; only under `/etc/netductor/secrets`.  
- Prefer fine-grained + least repos.  
- Rotate on schedule; `netductor update github-token clear repos` then set new.  
- Mirror URL uses `x-access-token` only for fetch; not exposed via API JSON.  
