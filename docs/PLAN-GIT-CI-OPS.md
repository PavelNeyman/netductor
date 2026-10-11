# Plan: Git / CI ops

**Status:** active · tip target 0.9.338+

## Go version

- **netductor:** `go 1.26` + CI `golang:1.26-bookworm` (current stable line; 1.27 available on go.dev — bump when docker hub ships `golang:1.27` widely).
- Other repos keep their own `go.mod` / container tags.

## Primary ↔ GitHub (required automation)

| Direction | When | Mechanism |
|--|--|--|
| **GH → primary** | manual / `sync` / `sync-all` | `MirrorFetch` + token read |
| **primary → GH** | after **push to bare** (post-receive) | `netductor git project mirror-push <name>` |
| **primary → GH** | after **successful VPS build** | `MirrorPush` in worker (non-fatal on error) |
| **Disable** | ops | `NETDUCTOR_GIT_MIRROR_PUSH=0` |

Requires GitHub token with **Contents: Read and write** on mirrored repos (`github_token_repos` or releases fallback).

## Phases

| P | Item | Status |
|--|--|--|
| P0 | Go bump + CI image | 1.26 this drop |
| P1 | Docker cleanup | done 0.9.337 |
| P2 | Queue worker on API | done 0.9.336 |
| P3 | Project card last build | done 0.9.337 |
| P4 | Mirror primary→GH auto | this drop |
| P5 | Lab e2e | OPEN_ITEMS |
| P6 | Cancel running + docker kill | 0.9.339 |
| P7 | Timers sync-all 6h + cleanup weekly | 0.9.339 |
| P8 | TG/Web last VPS + cancel | 0.9.339 |

## CLI

```bash
netductor git project mirror-push <name>
netductor git project mirror-push all
```
