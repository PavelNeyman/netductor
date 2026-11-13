# PLAN: Per-operator multi uplink A/B (S→P) — profile matrix

**Date:** 2026-11-13 (rev: multi-profile matrix)  
**Goal:** Operator gets **many** subscription URIs; each steers **only that session** to a different **secondary→primary** underlay. Family users stay on default uplink. After soak: keep winners, delete the rest.

**Non-goals:** Change client→S default Reality for everyone. Replace SP/wstunnel. DNS SERVFAIL work.

---

## 1. UX

One operator subscription, many nodes (names illustrative):

| Profile name | S→P underlay |
|--------------|----------------|
| `nd-up-default` | Current (Reality + mux) — baseline |
| `nd-up-nomux` | Reality same path, **multiplex off** |
| `nd-up-wg` | WireGuard UDP S↔P (plain) |
| `nd-up-wg-tcp` | WG over TCP wrapper if available / fallback path |
| `nd-up-anytls` | AnyTLS S→P |
| `nd-up-xhttp` | VLESS (or equivalent) + **XHTTP** transport + TLS |
| `nd-up-ws` | VLESS + **WebSocket** + TLS |
| `nd-up-grpc` | VLESS + **gRPC** + TLS |
| `nd-up-ssh` | SSH tunnel (lab only) |
| `nd-up-tsunami` | **Optional / experimental** — only if binary supports it |

In client: switch profile → watch speed/stability when “тупит”. No global apply for family.

---

## 2. Architecture

```text
                    ┌─ inbound-default ─► uplink-default (today)
Client (operator) ─┼─ inbound-nomux ───► uplink-nomux
                   ├─ inbound-wg ──────► uplink-wg
                   ├─ inbound-anytls ──► uplink-anytls
                   ├─ inbound-xhttp ───► uplink-xhttp
                   ├─ …                ► …
Family ────────────┴─ inbound-default only (no exp URIs in their sub)
```

- **One shared client face strategy:** either  
  - **A (preferred):** dedicated `listen_port` (or unique path/SNI) per exp inbound, URI encodes that port/path; or  
  - **B:** single :443 + route by special UUID per profile (messier for “switch in client”).  
- Prefer **A**: each profile = clear inbound tag → fixed outbound.

Primary terminates each underlay (WG peers, Reality/TLS inbounds for S-only, SSH, etc.). **Not** published as consumer VPN profiles.

---

## 3. Feasibility matrix (honest)

| Underlay | In sing-box stack | Effort S→P | Notes |
|----------|-------------------|------------|--------|
| default (Reality+mux) | Yes | 0 | Baseline |
| Reality **no mux** | Yes | Low | Best first A/B vs mux EOF theory |
| **WireGuard** UDP | Yes | Low–med | Best inter-VPS candidate; separate iface from SP |
| WG over TCP | Partial / external | Med | Only if tool exists; else skip v1 |
| **AnyTLS** | sing-box yes | Med | TLS camouflage S→P |
| **VLESS + WS + TLS** | Yes | Med | “Web-like”; need cert/SNI on P |
| **VLESS + gRPC + TLS** | Yes | Med | Same |
| **VLESS + XHTTP + TLS** | Xray-strong; sing-box varies by version | Med–high | Confirm pin supports outbound/inbound |
| **SSH** | `ssh` outbound or external | Low lab | CPU/MTU; **operator-only**, never promote |
| **TSUNAMI** | Proposal / fork — **not** stable mainline | High / defer | Enable only if build has it; else stub URI omitted |
| Hysteria2 / TUIC | Yes | Med | UDP; add if WG proves UDP path is fine |

**Ship matrix v1 (implement for real):**  
`default | nomux | wg | anytls | ws | grpc | ssh`  

**v1.1 if pin allows:** `xhttp`  

**Explicit defer:** `tsunami` until upstream support; `wg-tcp` unless a clean module exists.

Flag: `NETDUCTOR_UPLINK_AB=1` + allowlist users. Default **off** → generate zero exp inbounds.

---

## 4. Config shape (secondary)

Conceptual:

```json
"inbounds": [
  { "tag": "in-default", ... },
  { "tag": "in-nomux", "listen_port": 9444, ... },
  { "tag": "in-wg-steer", "listen_port": 9445, ... },
  ...
],
"outbounds": [
  { "tag": "uplink-default", ... multiplex ... },
  { "tag": "uplink-nomux", ... no multiplex ... },
  { "tag": "uplink-wg", "type": "wireguard", ... },
  { "tag": "uplink-anytls", ... },
  { "tag": "uplink-ws", ... },
  { "tag": "uplink-grpc", ... },
  { "tag": "uplink-ssh", "type": "ssh", ... }
],
"route": {
  "rules": [
    { "inbound": ["in-nomux"], "outbound": "uplink-nomux" },
    { "inbound": ["in-wg-steer"], "outbound": "uplink-wg" },
    ...
  ]
}
```

Steer inbounds can be minimal VLESS (same UUID as operator or dedicated test UUID) — only purpose is **select outbound**.

Ports: allocate a small range (e.g. 9444–9455) on S, firewall **not** world-open if possible (allow client IPs / or accept that Reality face is already public and exp ports are operator-risk — document).

---

## 5. Subscription

For allowlisted users only, append one link per **enabled** exp profile:

```text
# baseline (existing)
vless://…#nd-up-default
vless://…@S:9444…#nd-up-nomux
vless://…@S:9445…#nd-up-wg
...
```

Non-allowlisted: unchanged body (no exp ports leaked).

CLI: `netductor vpn uplink-ab list|enable|disable <id>` for ops without full regenerate guesswork.

---

## 6. Primary

| Exp | Primary needs |
|-----|----------------|
| nomux / ws / grpc / xhttp / anytls | Matching inbound(s) or single multi-inbound; TLS/Reality material |
| wg | `wg` iface + peer S, forwarding/NAT for user plane |
| ssh | `sshd` restricted user / key, no password; tunnel only |

Keep SP `nd-svc-*` **untouched**.

---

## 7. Phases

### Phase 0 — measure
SSH/TCP and current uplink goodput (inform priority).

### Phase 1 — harness
- Flag + allowlist  
- Port range + route table codegen  
- Sub emits N URIs  
- `uplink-ab status`  

### Phase 2 — fill underlays (parallel ok)
Order of implementation (not test order):

1. nomux  
2. wg  
3. ws + grpc (+ TLS)  
4. anytls  
5. ssh (lab)  
6. xhttp if supported  
7. tsunami — **gate on binary feature**

### Phase 3 — operator soak
Switch profiles when path degrades; log which profile was up.

### Phase 4 — rollback
- `NETDUCTOR_UPLINK_AB=0` removes exp inbounds/URIs on next apply  
- Delete dead primary peers  
- Optionally promote one winner to **default** uplink (separate change)

---

## 8. Risks

| Risk | Mitigation |
|------|------------|
| Config size / sing-box complexity | Cap matrix; disable types via file list |
| Open exp ports | Allowlist users only; optional firewall to known client nets |
| SSH CPU | Lab profile only |
| TSUNAMI instability | Don’t emit URI if unsupported |
| Operator confuses family phone | Exp only on allowlisted **account** subscription |

---

## 9. Acceptance

- Family sub identical to pre-flag.  
- Operator sub shows all **enabled** profiles; each maps to distinct underlay.  
- Flag off → clean default-only config.  
- Tests: golden JSON with N routes; allowlist filter on sub body.

---

## 10. Code map

| Area | Touch |
|------|--------|
| Generate S | `internal/vpn/secondary_box.go` (+ small `uplink_ab.go`) |
| Generate P | apply / inbounds / wg provision |
| Sub | `internal/vpn/subscription.go` |
| Flag | env + `/etc/netductor/uplink_ab.json` (enabled types list) |
| CLI | `vpn uplink-ab …` |

**Principle:** raise the **matrix harness** once; underlays are plugins in a list — easy to delete after soak.


---

## Implementation status

| Item | Status |
|------|--------|
| Harness + allowlist + URIs | **done** |
| Secondary steer inbounds 9444+ | **done** |
| Primary terminators ws **10443** / grpc **10444** / anytls **10445** | **done** (ApplyConfig) |
| WG keys auto-gen + primary `endpoints` + secondary outbound | **done** when `wg` in profiles |
| nomux | **done** (S→P existing :443, no mux) |
| ssh | needs `ssh_user`+`ssh_host` in json |
| CLI `vpn uplink-ab status` | **done** |
| Example | `docs/examples/uplink_ab.json` |

Enable: `enabled:true` in `/etc/netductor/uplink_ab.json` or `NETDUCTOR_UPLINK_AB=1` → `vpn apply` / secondary config rewrite + sub refresh. Open firewall for 9444+ on S and 10443–10445/UDP 51820 on P as needed.
