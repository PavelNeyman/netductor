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
