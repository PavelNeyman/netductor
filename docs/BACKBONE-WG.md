# Backbone WireGuard / AmneziaWG — design spike

**Status:** design only (no production code yet)  
**Related incident:** secondary→primary VLESS uplink mux timeouts while primary OS/API stayed up (2026-09)  
**Related code today:** `internal/vpn/secondary_box.go` (uplink VLESS+mux, no vision), `internal/secondary/agent.go` (uplink probe + sing-box restart watchdog)

---

## 1. Problem

Client path is healthy enough when Reality holds, but **control and transit between primary (EU) and secondary (RU)** share the same **VLESS Reality + multiplex** uplink:

| Symptom | Interpretation |
|---------|----------------|
| Secondary logs `outbound/vless[uplink]: timeout ~60s` | Stuck mux / middlebox / DPI on the **service** tunnel |
| Primary TG bot + API still work | Primary host alive; problem is **path secondary→primary:443**, not “server dead” |
| Watchdog restarts sing-box after 3 failed probes | Mitigates stuck state; does **not** give an independent control plane |

Client-facing Reality on secondary must stay. Backbone is **not** a replacement for end-user VLESS.

---

## 2. Goals / non-goals

### Goals

1. **Independent primary↔secondary channel** for:
   - agent heartbeat / commands (optionally prefer backbone IP)
   - backup pull / recovery fetch
   - future: metrics, cert material, light sync  
2. Survive periods when **Reality uplink is degraded** without killing user sessions on secondary inbounds (when possible).  
3. Optional **AmneziaWG (AWG)** if plain WG is filtered on a given path (spike-only until proven).

### Non-goals

- Replace client VPN with WG (carriers often block classic WG; already rejected as “admin-only public WG”).  
- Expose WG to the internet for operator laptops (SSH 52222 + keys remain day-2 access).  
- Full mesh between OpenWrt edges (edges stay agent→primary mTLS :8789).  
- CDN/XHTTP entry (separate idea in [OPEN_ITEMS.md](OPEN_ITEMS.md)).

---

## 3. Topology (recommended)

```
[Clients] --VLESS/Reality--> [Secondary RU] --VLESS mux--> [Primary EU]   // user data (as today)
                                 |                            ^
                                 +-------- backbone WG/AWG ---+   // service plane only
```

- **Listen:** primary only (EU), single UDP port (default proposal **51820**, configurable).  
- **Peer:** secondary only (one peer in v1).  
- **Addresses:** e.g. `10.87.0.1/30` primary, `10.87.0.2/30` secondary (private, not routed to clients).  
- **AllowedIPs on secondary:** only `10.87.0.1/32` (or primary backbone /30) — **no full-tunnel**, no steal of default route.  
- **AllowedIPs on primary:** `10.87.0.2/32`.  
- **Firewall:** UDP port open **only** from secondary public IP (nft/ufw allowlist); fail closed if IP unknown after rotate.

User traffic continues: secondary inbound → `uplink` VLESS → primary. Backbone is **orthogonal**.

---

## 4. What rides on backbone (v1)

| Traffic | Prefer backbone? | Notes |
|---------|------------------|--------|
| Secondary agent HTTP to primary `:8789` / API | **Yes** if up | Destination `https://10.87.0.1:8789` or keep public host but route via WG table |
| Recovery / backup pull secondary→primary | **Yes** when armed | Same allowlist as today + source IP backbone |
| Client VLESS uplink | **No** | Keep Reality path |
| Operator SSH | **No** | 52222 as today |
| Edge OpenWrt agents | **No** | Still enroll to primary public/mTLS |

**Routing policy on secondary:** policy rule “to 10.87.0.1 lookup backbone” or WG AllowedIPs alone (simplest).

---

## 5. WG vs AmneziaWG

| | **WireGuard** | **AmneziaWG (AWG)** |
|--|---------------|---------------------|
| Maturity | High; in kernel / `wg-quick` | Userspace/kernel module fork; extra packages |
| Obfuscation | None (easy DPI fingerprint) | Junk packets / header mods — may help mobile/RU path |
| Ops cost | Low | Higher (install, version pin, less “boring”) |
| Recommendation | **Default for spike and v1** | **Fallback profile** if WG handshake fails from secondary |

Spike acceptance: if plain WG primary↔secondary stays up for 24–48h under load while Reality uplink flakes, AWG is optional. If WG never handshakes from secondary, try AWG before abandoning backbone idea.

---

## 6. Keying & lifecycle

1. Primary install (or `netductor backbone init`): generate server keypair + peer slot; store under `/etc/netductor/backbone/` (0600).  
2. Secondary provision: inject peer private key + endpoint `primary_public_ip:51820` via existing Mac-direct deploy (same channel as agent cert).  
3. Rotate: regenerate peer keys; push via deploy; old peer removed.  
4. **Never** put backbone private keys in TG or client profiles.

Backup: include `/etc/netductor/backbone/` in component list when backbone enabled (config only, not kernel modules).

---

## 7. Failure modes

| Case | Behaviour |
|------|-----------|
| Backbone down, Reality up | User VPN OK; agent falls back to public primary URL (today’s path) |
| Reality uplink stuck, backbone up | Agent + backup via backbone; user may still suffer until uplink watchdog restarts sing-box |
| Both down | Same as today — secondary offline alerts |
| Primary IP change | Endpoint update on secondary (redeploy / node registry IP) |

**Fail-open for users:** backbone must not blackhole client traffic.  
**Fail-open for agent:** try backbone, then public mTLS URL.

---

## 8. Relation to h2mux A/B (next spike)

Today uplink uses sing-box **multiplex** without vision (`apply.go` / `secondary_box.go`). Stuck mux was the incident pain point.

**h2mux A/B** (separate work item):

- A: current multiplex settings  
- B: disable mux **or** alternate transport (e.g. h2) on uplink only  
- Measure: timeout rate secondary logs, Speedtest via secondary, agent uplink_ok streak  

Backbone **reduces urgency** of mux perfection for control plane but **does not replace** fixing user-path uplink quality.

---

## 9. Implementation sketch (when approved)

1. `internal/backbone` — generate configs, render `wg0` or sing-box wireguard outbound/inbound if we standardize on sing-box only.  
2. Prefer **kernel WG on Debian** primary/secondary for boring reliability; sing-box WG only if we must unify stack.  
3. CLI: `netductor backbone status|init|show`  
4. Deploy: Mac `DeploySecondary` optional `Backbone: true` (default on after soak).  
5. Doctor: handshake, last handshake age, transfer, endpoint reachability.  
6. TG/Web: status only (no key export).  

**Out of scope until spike OK:** AWG packages in install path, multi-peer, edge nodes on backbone.

---

## 10. Spike checklist (manual)

On a maintenance window:

1. [ ] Install `wireguard` on primary + secondary.  
2. [ ] `/30` addresses; UDP 51820 allowlist secondary→primary.  
3. [ ] `ping 10.87.0.1` from secondary.  
4. [ ] Point agent env `NETDUCTOR_PRIMARY_URL=https://10.87.0.1:8789` (or route only) temporarily; confirm heartbeat.  
5. [ ] Force Reality uplink stress (or wait for natural flake); confirm agent stays green via backbone.  
6. [ ] Confirm client VLESS still only uses Reality uplink (tcpdump/tag).  
7. [ ] If step 3 fails from mobile-like path: trial AWG once; document result in this file §5.

---

## 11. Decision log

| Date | Decision |
|------|----------|
| 2026-09 | Client public WG rejected (blocks). |
| 2026-09 | Uplink = VLESS+mux no vision; watchdog restart sing-box. |
| 2026-09-26 | This doc: backbone WG default, AWG optional, service plane only. |

