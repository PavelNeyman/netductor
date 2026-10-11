# Plan: Git / CI ops (post 0.9.336)

**Status:** active  
**Context:** GH mirror, VPS bare git + sequential build queue, dual tokens, workflow `.github/workflows/<Name>.yml`.

## Goals

1. **Go toolchain parity** — netductor on **Go 1.25** (same major as larvatus; M4tg may stay 1.26).
2. **Disk hygiene** on primary (docker CI images).
3. **Queue resilience** after reboot; observability.
4. **Operator UX:** last build status on project card; GH↔VPS policy.
5. **Secrets:** workflow PAT only when pushing yaml; day-2 tokens in `/etc/netductor/secrets`.

## Non-goals

- SonarQube on primary · parallel VPS builds · host=mac on VPS.

---

## Phases

### P0 — Go 1.25 + CI image ✅ / in this drop

| Item | Action |
|--|--|
| `go.mod` | `go 1.25` |
| `.github/workflows/netductor.yml` | `golang:1.25-bookworm`, `GOTOOLCHAIN=auto`, `go mod download` |
| Release | rebuild all assets after bump |

### P1 — Docker disk

| Item | Action |
|--|--|
| `netductor cleanup` | Detect + prune dangling images / unused CI containers (apply) |
| Doctor | optional WARN disk |
| Ops | run cleanup after large build-all |

### P2 — Queue resilience

| Item | Action |
|--|--|
| Worker on API start | already 0.9.336 |
| Reboot test | pending jobs resume |
| Cancel running | optional later |

### P3 — Project card status

| Item | Action |
|--|--|
| `ProjectDetail` | last VPS job from queue JSON |
| TG/Web | show status line |

### P4 — GH ↔ VPS policy

| Item | Action |
|--|--|
| Docs | primary-first vs GH-first + sync |
| Timer | optional `sync-all` (off by default) |

### P5 — Lab

OpenWrt / Tapo / dual-node — OPEN_ITEMS.

## Acceptance

- [ ] netductor tip `go 1.25` + CI image 1.25
- [ ] cleanup can free docker leftovers
- [ ] build-all queues; one worker
- [ ] project card last build
- [ ] this plan linked from SELFHOST-GIT

## Execute order

1. P0  2. P1  3. P3  4. P2/P4 follow-up  
