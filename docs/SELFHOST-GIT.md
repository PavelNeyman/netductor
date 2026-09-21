# Self-hosted Git (netductor thin model)

**Not a forge.** Single-operator bare repos + control from netductor.

## Status (v0.8.30)

| Action | CLI | API | Admin | TG |
|--------|-----|-----|-------|-----|
| List | `git list` | `GET /api/git/repos` | ✅ | ✅ |
| Init | `git init <name>` | `POST /api/git/repos` | ✅ | — (CLI) |
| Delete | `git delete <name>` | `DELETE /api/git/repos?name=` | ✅ | ✅ |
| Log | `git log <name> [n]` | `GET /api/git/log` | ✅ | ✅ |
| Show/diff | `git show <name> [rev]` | `GET /api/git/show` | ✅ | ✅ |
| Pipelines | `git pipelines` | `GET /api/git/pipelines` | ✅ | ✅ |
| Run pipeline | `git pipeline <repo> <script>` | `POST /api/git/pipeline` | ✅ | ✅ |
| GHA-subset workflow | `git workflow <repo> [yml]` | `POST /api/git/workflow` | — | — |

Paths:

- Repos: `/var/lib/netductor/git/<name>.git` (`NETDUCTOR_GIT_ROOT`)
- Pipelines: `/etc/netductor/git-pipelines/` (`NETDUCTOR_GIT_PIPELINES`)
- Samples auto-created: `echo-ok`, `go-test`

## Push from Mac

```bash
netductor git init netductor   # on primary
git remote add vps ssh://root@PRIMARY:52222/var/lib/netductor/git/netductor.git
git push -u vps main
```

On push, `hooks/post-receive` runs `netductor git pipeline <repo> $NETDUCTOR_GIT_PIPELINE` if that env is set on the server (e.g. in systemd or `/etc/environment`).

## CI model

| Approach | When |
|----------|------|
| **Pipeline scripts** in `git-pipelines/` | Default |
| **post-receive** + `NETDUCTOR_GIT_PIPELINE` | Auto on push |
| Woodpecker / Forgejo | Only if you later want a full forge |

## Registry (local OCI)

Thin **distribution/registry:2** on `127.0.0.1:5000` (not public Harbor).

| Action | CLI | API | Admin | TG |
|--------|-----|-----|-------|-----|
| Status | `registry status` | `GET /api/registry/status` | ✅ | ✅ |
| Ensure | `registry ensure` | `POST /api/registry/ensure` | ✅ | ✅ |
| Crane | `registry crane` | `POST /api/registry/crane` | ✅ | ✅ |
| Catalog | `registry catalog` | `GET /api/registry/catalog` | ✅ | ✅ |
| Stop | `registry stop` | `POST /api/registry/stop` | ✅ | ✅ |

```bash
netductor registry ensure
netductor registry crane
# pipeline with Dockerfile:
netductor git pipeline myapp oci-push
# or:
docker build -t 127.0.0.1:5000/myapp:latest .
crane push 127.0.0.1:5000/myapp:latest 127.0.0.1:5000/myapp:latest
```

Env: `NETDUCTOR_REGISTRY_ADDR` (default `127.0.0.1:5000`), `NETDUCTOR_REGISTRY_DATA`.


## Security

- No public HTTP git; SSH key-only (port 52222)
- API/Admin/TG require operator session / bot ACL


## GitHub Actions–subset workflows

Not full Actions. **Supported:** `jobs.*.steps[].run`, `env`, `working-directory`, `shell`.  
**Skipped:** `uses:`, `services`, `matrix`, marketplace actions.

```bash
# in repo: .github/workflows/ci.yml  (see docs/examples/ci.gha-subset.yml)
netductor git workflow myapp
netductor git workflow myapp .github/workflows/ci.yml
```

Auto on push (primary):

```bash
# prefer workflow over shell pipeline
export NETDUCTOR_GIT_WORKFLOW=1
# or explicit path relative to repo root:
# export NETDUCTOR_GIT_WORKFLOW=.github/workflows/ci.yml
```

On **GitHub**, the same YAML with only `run:` steps works as a normal workflow.  
**Policy:** target DSL is **GitHub Actions subset only** (popular + same file on GitHub). **GitLab CI** uses a different schema (`.gitlab-ci.yml`, `script:`) — not the same standard; porting requires a separate file or converter.

