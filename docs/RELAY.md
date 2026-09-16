# RU secondary VPS (legacy name: relay)

> **Operator name is secondary.** CLI/API still accept `relay` as an alias (`netductor secondary` ≡ `netductor relay`).  
> State path migrates `relay/` → `secondary/`. Prefer **secondary** in new docs and TG UI.


Chain for **mobile whitelist**:

```text
Phone  --VLESS Reality (SNI ya.ru)-->  RU relay  --VLESS Reality-->  primary (abroad)  --> Internet
Home   --VLESS Reality-------------->  primary (abroad)  --> Internet
```

## Hardware / network (RU hop)

| Item | Minimum | Recommended |
|------|---------|-------------|
| CPU | 1 vCPU | 1–2 vCPU |
| RAM | **512 MB** | **1 GB** |
| Disk | 10 GB | 15–20 GB |
| OS | Debian 12/13 | same |
| IP | Public **IPv4 in Russia** | Prefer **Yandex Cloud, VK Cloud, Timeweb, Selectel** (better whitelist chance) |
| Ports | **443/tcp** inbound open | Outbound **443** to primary (abroad) must work |
| Bandwidth | ~100 Mbps | Scale with users |

Relay runs **only sing-box** (no Blocky/API/Telegram/Lampac required).

## Automated setup

### On **primary** (abroad)

```bash
netductor vpn list
netductor relay export -o bundle.json --sni ya.ru
scp bundle.json root@RU_VPS:/root/
```

Creates core user `relay-uplink` (not for humans) and the join file.

### On new **RU** VPS

```bash
wget -qO /usr/local/bin/netductor \
  https://github.com/PavelNeyman/netductor/releases/download/v0.7.0-dev/netductor-linux-amd64
chmod 755 /usr/local/bin/netductor
netductor relay join /root/bundle.json
netductor relay links
systemctl is-active sing-box
```

### Clients

| Network | Where to get the link |
|---------|------------------------|
| Home | Core: `netductor vpn link NAME` |
| Mobile | RU: `netductor relay links` (also `/var/lib/netductor/relay/clients/`) |

Mobile URI uses **RU IP** and **relay Reality pbk/sid**; UUID matches the core user name.

## Re-sync after new VPN users

```bash
# core
netductor relay export -o bundle.json
# RU
netductor relay join /root/bundle.json
```

## Security

- Never distribute `relay-uplink`.
- Treat `bundle.json` as secret (`scp` only).
- Firewall on RU: 443/tcp public; SSH restricted.

## Managed agent (no SSH after join)

After `relay join`, RU runs `netductor-relay-agent`:

- Heartbeat → core `:8788` (`NETDUCTOR_RELAY_API`)
- Auto-pulls user list when core VPN users change
- Core shows status: `netductor secondary status  # alias: relay status` / Admin Relay / TG Relay
- Mobile links: `GET /api/relay/links` or TG (uses last reported IP+pbk)

Open on **core** firewall: **8788/tcp** from the RU IP (or world if needed).

## Provision from core (preferred)

Do **not** SSH into the RU VPS yourself. On core (or TG **Enroll secondary**):

```bash
netductor relay provision --host 92.x.x.x --user root --password '…' --sni ya.ru
```

Core will:

1. SSH with the one-time password  
2. Install the **same** SSH public key as on core (`/root/.ssh/id_ed25519.pub`)  
3. Disable password authentication  
4. Install `netductor`, join the relay bundle, start agent  

Afterwards: manage via TG/Admin/CLI only. Emergency SSH: same key as core.

## OpenWrt / clients: primary entry = secondary

Recommended:

1. **All clients** (phones, OpenWrt) use **secondary VLESS** as the only peer.
2. **Secondary** does split routing: RU domains/private → `direct`, else → core uplink.
3. **Fallback** on the router: if secondary `:443` is down, use WAN (ISP) without VPN — implement via `mwan3` / hotplug or agent healthcheck.
4. **Optional second peer (core)** only as manual emergency: traffic then hits foreign IP directly and may face DPI/whitelist issues on mobile.

### Why not dual-peer by default

A second peer to core is useful only when the operator accepts that the path may be unstable under whitelist. Prefer fixing relay availability over dual-peer.

### Client tunables (from field profiles)

- Reality fingerprint: `firefox` (or `chrome`)
- DNS for RU suffixes via `77.88.8.8`, else `1.1.1.1`
- Prefer IPv4-only on mobile paths
- Connectivity check interval ~5m (observatory) — future agent work
