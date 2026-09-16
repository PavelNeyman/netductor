# Open items

## Locked decisions
- [x] Subscription removed
- [x] Fleet primary (abroad) / secondary (RU) — not active-active equals
- [x] No VPN load-balancer VPS
- [x] SSH: password first login only, then key-only both nodes
- [x] Lampac localhost; prefer secondary
- [x] TG active on primary; standby via SOCKS→primary
- [x] TG: navigation under message; actions in body

## Open
- [ ] OpenWrt / MikroTik real hardware e2e
- [ ] Domain + HTTPS (optional)
- [ ] Path B end-user bot (idea only)

## Handoff
[AGENT_HANDOFF.md](AGENT_HANDOFF.md)


## Post-0.7.3 (backlog)

- [ ] Import redirect over **HTTPS** (Reality fallback on :443 or dedicated cert) so TG buttons are `https://` not `http://IP`
- [ ] Optional domain for `NETDUCTOR_REDIRECT_BASE` (`http://netductor.work.gd` / later real domain)
- [ ] Full mTLS client cert **provisioning** into secondary agent install path (CA already issues)
- [ ] LE for admin UI when non-rate-limited domain available; drop self-signed
- [ ] VPS reinstall checklist: redirect unit + bot env + mTLS + domains
- [ ] Secondary: confirm Blocky not public; only primary runs TG bot active
- [ ] Doctor check: `netductor-redirect` active on primary; `/healthz` on :80
