# Guest Wi‑Fi on OpenWrt (plan)

Status: **v1 agent implementation (0.8.25)** — nft + captive redirect + VPN bypass; hardware e2e pending · not all sites need guest · opt-in in edge wizard / template.


## Locked decisions (2026-09-21)

5. **Default grant duration: 10 minutes** (QR and desk default). Staff may choose up to **24 hours** when granting by code on desk.


1. **Layout: only separate Guest SSID** — no band-split preset. Private keeps 2.4 and/or 5 as today (cameras/IoT often need 2.4). Guest is an **extra** `wifi-iface` on chosen radio(s), default **2.4** if present, optional also on 5.
2. **Primary cashier UX: short session code + Grant QR**; device list is fallback (online/pending only, not years of MACs).
3. **Authorization: MAC allow-list + TTL** (default 24h), not daily Wi‑Fi PSK rotation.
4. **Stable Join QR** for association; internet only after grant.

## Problem

At a shop (or home office) the operator sometimes needs to **lend internet briefly** (e.g. customer pays via phone). Requirements:

1. Guest traffic: **internet only**, no access to LAN / cameras / private Wi‑Fi.
2. Guest traffic: **direct via ISP**, **not** through site VPN (VLESS/secondary).
3. Not every OpenWrt site enables this — **opt-in** in provision wizard / site template.
4. Prefer **hidden SSID** + connect via **QR**.
5. After a short window (default **10 minutes**, max **24h**) the device must **stop** using the internet unless explicitly extended.
6. Re-grant must be possible **without** Telegram bot and **without** logging into LuCI — by a **seller on the floor** (simple local action / QR).
7. Ideal UX: one **stable customer QR** for joining; access lifetime is separate from Wi‑Fi password rotation.

## Why “rotate Wi‑Fi password but keep the same QR” is a bad primary design

Standard Wi‑Fi QR (`WIFI:T:WPA;S:…;P:password;H:true;;`) **embeds the PSK**.  
If the PSK changes every 24h, the QR **must** change — unless the QR is not a Wi‑Fi QR but a **URL** to a page that shows the current credentials (useless for a guest’s phone off-LAN).

**Chosen model:** stabilize **association** (join network); control **authorization** (internet forwarding) by **MAC allow-list + TTL**.

| Layer | What it does |
|-------|----------------|
| Wi‑Fi join | Hidden guest SSID + long-lived PSK (or open+portal later). QR for join stays stable. |
| Internet | Only MACs on allow-list with `expires_at` get forward to WAN. Others: captive / no WAN. |
| Staff | Local “desk” UI on **private LAN only** (PIN): pending devices → Grant 24h / Revoke / Extend. |

Password rotation of the guest PSK remains **optional hygiene** (e.g. monthly), not the daily lock mechanism.

## Network topology (OpenWrt)

### Layout (single path): dedicated Guest SSID

**Not** “give all of 2.4 to guests”. Private Wi‑Fi stays as configured (2.4 and/or 5) for staff, POS, cameras, IoT.

Guest is an **additional** AP interface:

- Interface/bridge `guest` / `br-guest`, subnet e.g. `192.168.50.0/24`, DHCP only on guest.
- New `wifi-iface` (e.g. `guest24`) → `network guest`, `mode ap`, `hidden 1`, WPA2/3 PSK.
- Default radio: **2.4 GHz** when the device has it (better range in a shop hall). Optional second guest iface on 5 GHz later if needed — still same L3 guest zone.
- Private SSIDs/keys unchanged.

`guest_layout: band_split` — **rejected** (would kick cameras/IoT off 2.4).

### Firewall (nft/fw4)

Zone `guest`:

- `guest → wan`: ACCEPT (only if MAC allowed — see below).
- `guest → lan`: REJECT / DROP.
- `lan → guest`: DROP (or limited DHCP helper only).
- Input on guest: ALLOW DHCP+DNS to router; REJECT SSH/UI from guest.

Implement MAC gate either:

- **fw4** `mac` rules / ipset + timer, or  
- **nft set** `guest_allowed { type ether_addr; flags timeout; }` with timeout = remaining TTL.

Without MAC in set: no forward to wan (optional redirect to portal HTTP).

### VPN bypass

Site VPN (agent outbound / policy routing) must **not** capture guest:

- Mark or table: traffic from `br-guest` / iif guest → **main / wan** only.
- Explicit `ip rule` / PBR: `iif br-guest lookup main` (or disable VPN for source `192.168.50.0/24`).
- Never send guest into VLESS/tun.

Doctor check: from a guest-associated test MAC, egress IP = ISP, not secondary/core.


## Short session code (captive) — detail

When a phone associates to guest and has **no** valid allow-list entry:

1. DHCP assigns an address; agent notes `mac`, `lease time`, optional hostname.
2. Agent creates a **session**: `{ code: "7K2", mac, created_at, expires_at: now+15m }`.
3. Captive portal (gateway HTTP) shows large text:
   - RU: «Код для продавца: **7K2**»
   - Optional: Grant QR encoding the same session (one-time token).
4. Cashier either:
   - types `7K2` on Guest Desk → sees that MAC → Grant 24h, or
   - scans Grant QR → same grant without typing.
5. Code is **not** the Wi‑Fi password. It only points at “this phone right now”.
6. Codes are short (3–4 chars, unambiguous alphabet), unique among **active** sessions; recycled after expiry.
7. Desk default view: **Pending / online now** sorted by newest — typically a handful, not historical thousands.

Historical MACs (years) are **not** shown on desk; optional admin log with retention (e.g. 30–90 days) only.

## Authorization model (24h)

```text
Device associates to guest Wi‑Fi (stable QR)
        │
        ▼
DHCP lease + MAC known to agent
        │
        ├─ MAC not in allow-list → DNS/HTTP portal “waiting / ask staff”
        │                          no WAN forward
        │
        └─ MAC in allow-list, now < expires_at → full WAN
                 expires → auto remove from set (nft timeout or agent cron)
```

**Grant:** `expires_at = now + 24h` (configurable 1h / 4h / 24h).  
**Extend:** same action again (+24h from now or from previous expiry — pick **from now** for simplicity).  
**Revoke:** delete MAC immediately.

### Seller without bot / LuCI

**Guest desk** — HTTP on **LAN IP only** (e.g. `http://192.168.1.1:7880/guest/`), not on guest net, not on WAN:

1. Staff phone on **private** Wi‑Fi opens desk (bookmark).
2. PIN (site-local, set at provision; stored hashed on device).
3. UI: list of recent guest MACs / hostnames / “pending portal”.
4. Buttons: **Grant 24h** · **Extend** · **Revoke**.
5. Optional: show **customer join QR** (stable) for printing.

**QR variants:**

| QR | Content | Who scans |
|----|---------|-----------|
| **Join** (stable) | `WIFI:…` guest SSID+PSK, `H:true` | Customer |
| **Desk** (optional) | URL `http://<lan-ip>:7880/guest/` | Staff (bookmark) |
| **Grant token** (optional v2) | one-time token URL on LAN that grants the **currently pending** MAC | Staff at counter |

v1: Join QR + desk buttons. v2: token QR if needed for speed.

### Captive portal (v1 light)

Minimal: agent serves a page on guest subnet gateway:80 “Access required — ask staff”.  
Full nodogsplash optional later — not required if nft simply blocks WAN until allow.

## Configuration (edge template / wizard)

Opt-in fields (site template + OpenWrt wizard):

```yaml
guest:
  enabled: false
  band: "2g"            # 2g | both — which radio(s) get guest AP (private unchanged)
  ssid: "Shop-Guest"
  hidden: true
  psk: "<generated>"    # long-lived; rotatable manually
  subnet: "192.168.50.0/24"
  default_grant_minutes: 10
  max_grant_minutes: 1440
  desk_pin: "<set>"
  desk_port: 7880
```

Wizard questions (only if “Enable guest Wi‑Fi?” = yes):

1. Enable guest? Hidden SSID? (default yes).  
2. Guest on 2.4 only or also 5 (private unchanged).  
3. Default grant TTL (24h).  
4. Desk PIN.  

Primary stores template per site; agent applies UCI + firewall + desk service.

## Agent responsibilities (`netductor-agent`)

1. Apply guest UCI (network, wireless, dhcp, firewall) idempotently.  
2. Maintain allow-list file e.g. `/etc/netductor/guest-allow.json` + sync nft set.  
3. Cron / loop: expire MACs; reload nft.  
4. Desk HTTP (LAN bind only): auth PIN, list leases+allow, grant/revoke.  
5. Generate join QR payload string for desk UI.  
6. Report guest stats in heartbeat (optional): allowed count, expires soon).

Primary / TG / Admin (later, optional): remote grant — **not** required for shop floor v1.

## Security notes

- Desk **never** on `0.0.0.0` WAN; bind LAN or `127.0.0.1` + reverse only from lan zone.  
- Guest cannot reach desk, SSH, LuCI, cameras.  
- PIN brute-force: rate limit + lockout.  
- PSK still secret; rotate if leaked (new join QR printed).  
- Isolation tested: guest ping lan host fail; guest curl WAN OK only when granted.

## Implementation phases

| Phase | Scope |
|-------|--------|
| **G0** | Docs + template fields + wizard flag only (no apply yet) |
| **G1** | UCI guest + isolation + VPN bypass | **done 0.8.25** |
| **G2** | MAC allow-list + TTL + nft sync | **done 0.8.25** |
| **G3** | Desk UI + join QR text | **done 0.8.23** |
| **G4** | Captive page + HTTP DNAT redirect | **done 0.8.25** |
| **G5** | Optional: TG/Admin remote grant; grant-token QR; band_split presets |

## Out of scope (for now)

- Writing guest traffic to NVR / content filter profiles.  
- Paid voucher system.  
- TUI full parity before G3 (CLI/agent first).  
- TUI general refactor (see deferred item).

## Acceptance scenarios

1. Guest joins via stable QR; **no** internet until grant.  
2. Staff grants 24h on desk; payment works; **no** access to `192.168.0.0/16` LAN.  
3. Traceroute/VPN check: guest egress = ISP, not tunnel.  
4. After TTL: internet stops without staff action.  
5. Second visit: staff Grant again — works.  
6. Site with `guest.enabled=false`: no guest iface, no desk port.
