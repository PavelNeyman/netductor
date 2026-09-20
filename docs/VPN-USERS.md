# VPN users

## Secondary must list every user UUID

After `vpn add` / apply, secondary agent pulls the user list. If a new user cannot connect via secondary IP:

- Force: `netductor secondary sync` then wait one heartbeat.
- Check `netductor fleet status` / `netductor secondary status`.

Canonical plane name is **secondary** (not `relay`).
