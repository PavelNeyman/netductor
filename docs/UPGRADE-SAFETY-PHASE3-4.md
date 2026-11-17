# Upgrade safety — Phase 3/4 vs release v0.9.756

## What runs on servers today

Published GitHub release: **v0.9.756**.  
`main` tip may be **ahead** of that tag (Phase 3 control role, sync-pull, Phase 4 node status). Until the next release is cut and applied, servers only run **0.9.756** behaviour.

## Safe update to the current *published* release

If primary is already on **0.9.756**, nothing required for release parity:

```bash
netductor version          # expect 0.9.756 (node)
```

If older:

```bash
# primary
netductor stack apply v0.9.756
# then each secondary (from primary)
netductor secondary status
netductor secondary cmd <id> upgrade:v0.9.756
```

Mac operator:

```bash
brew update && brew upgrade netductor-op   # or netductor-op update
hash -r && netductor-op version
```

Preflight already checks `upgrade.json` / `min_from` (CF-15). **0.9.756** has no binary floor that blocks skip-version from recent 0.9.x.

## Phase 3/4 code (on main, not yet a release)

| Feature | Behaviour if never configured | Risk on update when released |
|---------|-------------------------------|------------------------------|
| `control_role` file missing | **active**, writes allowed | none |
| `SERVERS=` empty | single `SERVER` as today | none |
| sync-timer not installed | no timer | none |
| `/api/node/*` | new routes only | none |

**Hardware / live cutover is not required to land code.** Defaults preserve single-control, single-egress operation.

## When the next batch release is cut (operator chooses)

1. Read release notes / this doc.
2. `stack apply v0.9.xxx` on primary (automatic migrations + host baseline convergence).
3. Upgrade secondary agents explicitly (`secondary cmd … upgrade:…`).
4. Optional: `control token-init`, standby, `SERVERS=` — only when you start HA testing.

Do **not** enable standby or dual control endpoints until you intend to test Phase 3.

## Rule

Land commits on `main` without a release every time. Cut a release at phase boundary or when a binary is required. Skip-version jumps must remain valid (migrations + CF-15).

## Related

- [RUNBOOK-CLUSTER-STATUS.md](RUNBOOK-CLUSTER-STATUS.md) — cluster / thin status after upgrade
- [RUNBOOK-CONTROL-CUTOVER.md](RUNBOOK-CONTROL-CUTOVER.md)


## After stack apply (multi-egress /sub)

`/sub` includes only secondaries with recent heartbeat **and** `singbox_ok=true`.
If profiles briefly disappear after upgrading primary only, wait ~30–60s for secondary agent heartbeat, or check:

```bash
netductor secondary status   # sb=true
```

Do **not** set `control_role=standby` on the sole control node.
