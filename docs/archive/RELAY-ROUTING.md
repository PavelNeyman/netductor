# Secondary routing (RU / non-RU)

> Legacy title: Secondary routing. Operator name: **secondary**.


RU domains + private → direct on secondary. Other client traffic → uplink to core. Optional exit-in for RU egress from abroad. See `docs/ru/RELAY-ROUTING.md`.

## geoip-ru

In addition to `domain_suffix`, secondary sing-box loads remote rule-set `geoip-ru` (SagerNet sing-geoip). IP ranges of RU → **direct**. Download detour is **uplink** so the database can be fetched via primary if GitHub is blocked from the RU VPS.
