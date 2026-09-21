# Self-hosted Git for netductor (single operator)

## Goal (clarified)

Not a forge (no multi-user web, PRs, issues, comments).

**Wanted:** private source on primary + control from **netductor** (CLI / Admin / TG later):

| Action | Enough? |
|--------|---------|
| List bare repos | yes |
| Create / delete bare repo | yes |
| Recent commits + message | yes |
| Diff for a commit | yes (`git show`) |
| Trigger a pipeline | yes |
| Full GitLab/Forgejo UI | **no** |

That is **bare git + thin wrappers**, not “write GitLab”.

---

## Stack by layer (facts, 2026)

### 1. Git storage — only `git`

- `git init --bare /var/lib/netductor/git/<name>.git`
- Access: existing SSH (port **52222**, operator key)
- Remote example: `ssh://root@PRIMARY:52222/var/lib/netductor/git/netductor.git`
- List/log/show/diff: pure **git** CLI (or go-git in-process later)

No alternative “lighter than git”.

### 2. CI/CD — not only heavy products

| Approach | Weight | Fits single operator? |
|----------|--------|------------------------|
| **`post-receive` hook** → script (`go test`, `build-release`) | Minimal | **Best default** |
| **Pipelight** (CLI-only pipelines, git hooks) | ~13 MB binary | Good if you want declared pipelines |
| **`act`** | Local GitHub Actions runner | Useful on Mac; not a server forge |
| **Woodpecker / Drone** | Server + agent + Docker | Light vs Jenkins, but **expects a forge OAuth** (Gitea/Forgejo/GitHub) — awkward with bare-only |
| Jenkins / GitLab CI | Heavy | No |

**Fact:** there is no widely used “CI daemon as simple as `git` binary” that watches bare repos without either hooks or a forge API. For one person, **hooks + netductor “Run pipeline”** (exec script, stream log) is the right shape.

### 3. Container registry

| Piece | Role |
|-------|------|
| **`registry:2`** (distribution) | Small OCI registry daemon (or skip and only push to GHCR privately) |
| **crane** / **skopeo** / **regctl** | CLI like git for tags, copy, inspect, delete — **no Docker daemon required** for most ops |

Harbor is the “big” product; not needed for personal images.

---

## Recommended architecture for netductor

```text
Mac ──SSH:52222──► primary
                    /var/lib/netductor/git/*.git   (bare)
                    hooks/post-receive → optional build script
                    netductor git list|log|show|pipeline
Admin/TG → same API
crane/skopeo → optional local registry or external
```

1. Implement thin **`netductor git …`** (and API) wrapping `git`.
2. Pipeline = named script under `/etc/netductor/git-pipelines/` or repo `hooks/`.
3. Add registry only when you actually publish images to yourself.

Do **not** pull Woodpecker until you adopt Forgejo/Gitea; without a forge it fights you.

---

## CLI status in tree

See `netductor git -h` (from v0.8.28): `init`, `list`, `log`, `show`.
