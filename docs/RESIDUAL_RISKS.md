# Residual risks (detailed) — v0.8.18

## 1. HTTP redirect (`redirect-serve`)

**What:** TG deep-link buttons may use `http://…/r?u=…` so clients open custom schemes.

**Risk:** Cleartext on the wire; open port if bound to `0.0.0.0:80`.

**Mitigation (0.8.18+):** Default listen **`127.0.0.1:80`**. Public HTTP only with explicit `-listen :80`. Prefer VPN path or HTTPS (`-https-listen` + certs) when domain exists.

## 2. Edge recovery HTTP (`:7879`)

**What:** One-time code form on the router for re-attach without full re-provision.

**Risk:** If bound to all interfaces and reachable from WAN, attacker on path could try codes (still need valid recovery/bootstrap secret).

**Mitigation (0.8.18+):**
- Prefer **private LAN IP** bind; if none → **127.0.0.1**, not `0.0.0.0`.
- **No auto-fallback** to public `:7879`.
- Clients: only loopback/private/link-local unless `NETDUCTOR_RECOVERY_ALLOW_ANY=1`.
- Disable: `NETDUCTOR_RECOVERY_HTTP=0`.

## 3. Agent plane `:8789` from internet

**What:** mTLS plane for edge/secondary behind NAT.

**Risk:** Port is probeable; without client cert handshake fails. Resource exhaustion if rate-limit insufficient.

**Mitigation:** mTLS required, 180/min/IP, ban after repeated 429, ufw allows 8789 (required). Optional future: fail2ban-style longer bans.

## 4. Footguns

| Env | Risk | Action |
|-----|------|--------|
| `NETDUCTOR_PLAIN_AGENT=1` | Plain `:8788` agent API | Never on prod; doctor FAIL |
| `NETDUCTOR_API_PUBLIC=1` + bind non-local without TLS | Refused; with TLS exposes admin | Only via VPN or strict ACL |
| `CLAIM_FIRST` / `NETDUCTOR_TG_CLAIM_FIRST=1` | First chatter becomes admin | Disable after bootstrap; doctor WARN |
| Long-lived edge bootstrap token | Device enroll | Prefer short recovery codes |

## 5. Product-public (encrypted by design)

VPN 443/4443/8443, SSH 22/52222 — not “bugs”. Keep Reality/HY2 params and key-only SSH.

## 6. Upgrade / remote cmds

Secondary `upgrade` is now **Go-only** (no `bash -c`). Operator-initiated `uci`/`reboot` still powerful — only for **approved** edge devices.
