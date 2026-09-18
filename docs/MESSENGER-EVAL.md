# Family messenger evaluation (2026-09-18)

Goal: self-hosted family chat with **voice**, preferably **video**, mobile clients, usable under RU constraints.

## Decision snapshot

| # | Project | Status | Notes |
|---|---------|--------|-------|
| 1 | **Snikket** | **candidate** | Own iOS/Android, A/V calls, family-oriented, Docker |
| 2 | **Tinode** | keep (blocked client in region?) | Go stack, 1:1 A/V; store apps may be geo-limited — recheck later |
| 3 | **Matrix + Element X** | fallback | Known stack; use if nothing lighter fits |
| 4 | **Guardyn** | **future watch** | Strong E2EE + A/V (SFrame); mobile was promised ~Q1 2026, still not in stores — revisit when iOS/Android ships |
| 5 | Seclettr | **out** | No iOS client |
| 6 | Stoat | think later | Discord/Slack-like; iOS TestFlight full / limited |
| 7 | Databag | looking | Lightweight; needs TURN for calls |
| 8 | Mqvi | think later | Voice/video heavy (LiveKit); Discord-ish |
| 9 | SimpleX | looking | Privacy-first; different UX |
| 10 | Delta Chat | **out** | Not a live A/V messenger substitute |

## Guardyn (watch list)

- Site: https://guardyn.co/
- GitHub: https://github.com/guardyn/guardyn
- Why interesting: self-host, PQ crypto, E2EE voice/video (SFrame), Flutter clients planned
- Blocker: no usable mobile store clients yet (as of 2026-09)
- Action: re-check releases/App Store/Play every few months; not for netductor addon until mobile exists

## Next

- Hands-on UI check: Snikket, Databag, SimpleX
- If pilot: prefer Snikket or Matrix behind VPN + coturn

## Pilot deployment (primary, no domain) — 2026-09-18

Deployed for hands-on testing (not automated in `netductor install`):

1. **Databag** — Docker on primary `:7000`, proxy on secondary `:7000`. Admin = env `ADMIN` via UI cog. Create users via invite. Weak without DNS/HTTPS for mobile.
2. **SimpleX SMP + XFTP** — systemd on primary `:5223` / `:5224`; socat proxies on secondary. **Use `2.27.118.70` host in app when client is on secondary VPN** (avoid hairpin to secondary IP).
3. **Snikket** — not deployed: needs domain + free :443 (conflict with VLESS Reality) or SNI front / separate VPS.

Details, firewall, test checklist: [AGENT_HANDOFF.md](AGENT_HANDOFF.md) § Family messenger pilot.
