# Shadowrocket and netductor

See also [ru/SHADOWROCKET.md](ru/SHADOWROCKET.md).

## Profiles

| File | Role |
|------|------|
| VLESS URI from TG bot | Prefer **secondary**, `flow=xtls-rprx-vision` |
| `nd-oc.conf` / SR Config | Mac + OpenConnect: LAN/corp + **RU/gov DIRECT** |
| `shadowrocket-routing.conf` | After `vpn client-config`: RU DIRECT + FINAL PROXY |

## Gosuslugi / banks

RU domains and `GEOIP,RU` → **DIRECT** (home ISP). Rest → PROXY.

App messages “disable VPN” may still appear (tun detection) — routing cannot fix that.

## Env

- `NETDUCTOR_REDIRECT_BASE` — optional HTTP base for deep-link import buttons (no hardcoded VPS IP).
- SR Config is delivered as a **Telegram document** even when redirect base is unset.
