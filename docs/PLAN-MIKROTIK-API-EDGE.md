# Plan: MikroTik API via edge agent (deferred)

Status: **optional design** — not implementing now.

## Idea

Raspberry Pi (OpenWrt agent) on LAN:

1. Talks to MikroTik **API** (LAN only, not WAN).
2. Reports status / applies small changes to primary over existing **mTLS :8789**.

Bootstrap remains Mac `DeploySite` (SSH + RSC). API path is day-2 only.

## Why

Richer MT state than RSC-only; no public MT API; fits MT+RPi sites.

## Security

- API service bind to LAN/bridge only
- Dedicated low-privilege user
- Credentials only on Pi (600)
- Never ship MT password to TG

## Non-goals

- Replace first-time RSC push
- Expose API on 0.0.0.0
