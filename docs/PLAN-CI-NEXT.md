# PLAN-CI-NEXT

Status: **implemented in 0.9.360**

| Item | Default |
|--|--|
| Step timeout | 45m (`NETDUCTOR_SHIP_STEP_TIMEOUT`) — only running steps |
| CI secrets | `/var/lib/netductor/ci-secrets/<project>/KEY` |
| CI cache | `/var/lib/netductor/ci-cache/` GOCACHE/GOMODCACHE/npm; prune 14d |
| Job log | `ship-jobs/<id>/ship.log` · `project job log` · TG 📜 |
| Auto ship | `on_push` / `on_tag` / `build_on_fetch` + `path_ignore` |
| Publish | SHA256SUMS next to artifacts |

Not in scope: GHA `uses:`, parallel matrix, PR checks UI.
