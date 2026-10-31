# Plan: APT Repository Guard

## Integration map (stage 0 research)

| Area | Existing | Guard use |
|------|----------|-----------|
| Server inventory | secondary registry, nodes | scan primary local + secondary via agent/jobs |
| Remote execution | secondary agent, deploy SSH (key-only) | prefer agent job; avoid ad-hoc SSH for scan |
| Findings | doctor checks, metrics | store under `/var/lib/netductor/aptguard/` |
| Notifications | `internal/notify`, TG | dedupe by finding code+server |
| UI | doctor, web badges | status in doctor + server card |
| Scheduling | systemd timers, metrics | timer 6h + post-deploy hook |
| Audit | install/audit patterns | remediation audit log |
| APT helpers | `internal/install/apt.go` waitAptLock | remediation only |
| Baseline | AptPinOK hoster pin | **keep separate**; Guard is sources policy |

## Principles

- Scan is **read-only** (no apt-get update in scan path).
- Remediation only after explicit admin confirm; backup + rollback.
- Doctor consumes Guard report; does not re-parse sources itself.

## Stages

0 Research — done (this doc)
1 Policy store + validate — code in `internal/aptguard`
2 Scanner .list + .sources — code in `internal/aptguard`
3 Findings store/history — next
4 Primary/Secondary host integration
5 Doctor wire-up
6 Scheduling 6h
7 TG + Web
8 Remediation (after read-only stable)
9 E2E

## Status

- [x] Stage 0 research map
- [x] Stage 1 DefaultPolicy + Validate
- [x] Stage 2 Scanner list + deb822 + ScanHost compare
- [x] Stage 3 findings store + history + DiffFindings
- [x] Stage 5 Doctor summary (primary local report)
- [x] CLI `netductor aptguard scan|status|policy`
- [x] Stage 4 Secondary cmd aptguard_scan + API queue
- [x] Stage 6 netductor-aptguard.timer (6h) via host-baseline
- [x] Stage 7 API GET/POST /api/aptguard/* + opcatalog (thin UI)
- [ ] Stage 8 Remediation
- [ ] Stage 9 E2E
