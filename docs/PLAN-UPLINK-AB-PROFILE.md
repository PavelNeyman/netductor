# PLAN: Per-operator uplink A/B (S→P) without affecting other users

**Date:** 2026-11-13  
**Goal:** Test alternate **secondary→primary** data-plane underlays while family users stay on the current uplink. Operator switches path from the **VPN client** via a second subscription profile (URI), not via global `vpn apply` for everyone.

**Non-goals:** Replace SP/wstunnel (service plane). Change client→S Reality face for all users. DNS SERVFAIL work (separate).

---

## 1. Context

| Layer | Today | A/B target |
|-------|--------|------------|
| Client → S | VLESS + Reality :443 (shared) | Unchanged for default users |
| **S → P (uplink)** | Single shared outbound (Reality + mux) | **Default** vs **exp** outbound |
| Sub profile `secondary` / `primary` / `both` | Which **entry** URIs (S and/or P) | **Not** which S→P underlay |
| Users | One secondary sing-box config | Route **by user or by inbound tag** |

Without code, “switch profile in client” only changes entry host, **not** S→P protocol.

---

## 2. Desired UX

Operator subscription contains at least:

| URI name (example) | Path |
|--------------------|------|
| `nd-secondary-…` | Current uplink (baseline) |
| `nd-uplink-wg` (or `…-nomux`, `…-ssh`) | Experimental S→P |

Switch node in client → only **that** session uses exp underlay. Everyone else: baseline.

Optional later: more exp slots (`nomux`, `hysteria`, …) — same pattern.

---

## 3. Design (recommended)

### 3.1 Outbounds on secondary

- Keep existing `uplink` (or current tag) as **default**.
- Add `uplink-exp` (configurable type):
  - **Phase B1:** WireGuard S↔P (user egress only; not SP).
  - **Phase B0 (fast):** same Reality host/port **without multiplex** (or mux mode off) — isolates mux EOF theory.
  - **Phase B2 (lab):** SSH / other — operator-only, not product default.

Primary must terminate the chosen underlay (WG peer, or existing Reality inbound already used for uplink).

### 3.2 Steering (pick one; prefer inbound)

**Option A — separate inbound (best for client profiles)**  
- Extra inbound on S: e.g. different `listen_port` or dedicated tag/SNI path only in exp URI.  
- Route: `inbound: [vless-exp] → outbound: uplink-exp`.  
- Default inbound → `uplink`.  
- Operator sub gets second VLESS link pointing at exp inbound.  
- **Pros:** Clear A/B in client; no reliance on UUID routing. **Cons:** Extra port/firewall hole (restrict to known peers if possible).

**Option B — auth_user / user list**  
- Route: users in `uplink_exp_users` (e.g. `operator`, `pavel`) → `uplink-exp`.  
- **Pros:** One :443. **Cons:** Cannot switch back to baseline without config change; bad A/B UX.

**Recommendation:** **Option A** for productized A/B; Option B only as one-shot debug.

### 3.3 Subscription

- Extend body generation for allowlisted names (operator / explicit flag `uplink_ab: true` on user record **or** env `NETDUCTOR_UPLINK_AB_USERS=operator,pavel`):
  - Always emit baseline secondary URI(s) as today.
  - Emit **one** exp URI when exp inbound is configured and healthy.
- Normal users: **no** exp URI (no leak of test endpoints).

### 3.4 Primary side

- WG: interface + peer for S, policy route **only** marked/user-plane traffic if needed (do not steal default route).  
- Or reuse existing uplink Reality inbound (already there for today’s S→P).  
- Do not bind exp termination to public client face in a way that opens new consumer profiles.

### 3.5 Safety

- Exp path **off** unless `NETDUCTOR_UPLINK_AB=1` (or feature file).  
- Default route for all non-exp traffic unchanged.  
- `relay-uplink` remains service identity for default uplink auth — not an end-user profile.  
- Document: compromised secondary can still claim users (existing trust note on `auth_user`).

---

## 4. Phases

### Phase 0 — measure (no multi-profile yet)

1. From S: TCP/SSH goodput to P vs current uplink goodput.  
2. Confirm whether underlay type matters before coding A/B.

### Phase 1 — code skeleton (Option A)

1. Config flags: exp outbound type + params; exp inbound port.  
2. Generate secondary box: second outbound + inbound + route.  
3. Subscription: second URI for allowlisted users only.  
4. CLI: `netductor vpn uplink-ab status` (default vs exp configured, users allowlist).  
5. Tests: generated JSON has two paths; non-allowlisted sub has no exp URI.

### Phase 2 — WG exp underlay

1. Provision WG S↔P for user plane (separate from SP `nd-svc-*`).  
2. Primary NAT/egress for traffic arriving via this WG.  
3. Soak with operator profile only.

### Phase 3 — optional variants

1. `uplink-exp` = Reality no-mux (if Phase 0 implicated mux).  
2. Kill switch: disable exp inbound via flag without full regenerate pain.

### Phase 4 — promote or drop

- If exp wins: optional migration of **default** uplink (separate change, all users).  
- If not: remove exp URI / keep flag off.

---

## 5. Explicitly not in v1

- Per-user UI in TG to “toggle uplink” for family accounts.  
- Automatic traffic split by domain across two underlays.  
- Replacing client→S Reality.  
- Using SP wstunnel as user data plane.

---

## 6. Acceptance

| Criterion | |
|-----------|--|
| Family users unchanged (URI + path) | |
| Operator can switch baseline ↔ exp in client | |
| Feature flag default off | |
| Tests for generate + sub allowlist | |
| Doctor/status shows which underlay exp uses | |

---

## 7. Implementation touchpoints (code map)

| Area | Files (indicative) |
|------|---------------------|
| Secondary generate | `internal/vpn/secondary_box.go` |
| Primary uplink peer / WG | `internal/vpn/` apply + svc/wg helpers as needed |
| Subscription URIs | `internal/vpn/subscription.go` |
| Flags / allowlist | env + optional user field |
| CLI | `cmd/netductor` vpn/channel-adjacent |
| Tests | secondary config golden + sub body |

---

## 8. Order of work

1. Phase 0 measurements (operator).  
2. Phase 1 skeleton + no-mux or WG behind flag.  
3. Operator-only soak.  
4. Decide promote vs drop — **no** silent default change.

**Related:** channel telemetry / incidents measure path health; this plan changes path **choice** for one client profile only.
