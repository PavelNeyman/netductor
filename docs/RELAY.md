# Secondary (VPN entry) — formerly “relay”

> **Canonical operator CLI: `netductor secondary` and `netductor fleet`.**
> There is **no** `netductor relay` command in the current binary.
> Internal code may still say `role=relay`, `relay-in`, `ExportRelayBundle` — same plane.
> Architecture: [FLEET.md](FLEET.md) · [PLAN-SECONDARY-VPN-ONLY.md](PLAN-SECONDARY-VPN-ONLY.md).

## Role

RU **secondary** = **VPN entry only** (VLESS/Reality + thin agent). Not a mirror of primary (no Lampac/bot/edge enroll).

Typical path under carrier whitelist:

```text
Phone  --VLESS Reality-->  RU secondary  --uplink-->  primary (abroad)  --> Internet
Home   --VLESS Reality-->  primary (or secondary, by profile)  --> Internet
```

## Hardware (RU hop)

| Item | Minimum | Recommended |
|------|---------|-------------|
| CPU | 1 vCPU | 1–2 vCPU |
| RAM | 512 MB | 1 GB |
| Disk | 10 GB | 15–20 GB |
| OS | Debian 12/13 | same |
| IP | Public IPv4 in Russia | Yandex Cloud, VK Cloud, Timeweb, Selectel (WL chance) |
| Ports | **443/tcp** inbound | Outbound 443 to primary |

## Setup (preferred)

From **primary**:

```bash
netductor fleet provision-secondary --host RU_IP --password '…' [--sni api.vk.me]
netductor secondary sync
netductor fleet status
```

Manual alternative:

```bash
# primary
netductor secondary export -o bundle.json --sni api.vk.me
scp bundle.json root@RU_VPS:/root/

# RU (binary v0.8.1)
export NETDUCTOR_VERSION=0.8.1
curl -fsSL https://raw.githubusercontent.com/PavelNeyman/netductor/main/bootstrap.sh | bash
netductor secondary join /root/bundle.json
netductor secondary links
systemctl is-active sing-box
```

Treat `bundle.json` as secret. After provision, SSH password on secondary is typically disabled (same key as primary).

## Day-2

```bash
netductor secondary sync      # push VPN users to secondary now
netductor secondary status
netductor secondary links
netductor fleet status
```

New VPN users: after `vpn add` / apply, wait for agent pull or force `secondary sync`.

## Clients

| Network | Link source |
|---------|-------------|
| Prefer WL / mobile | Links via secondary IP (`secondary links` / TG when secondary online) |
| Home | Core or secondary per profile — [CLIENT-PROFILES.md](CLIENT-PROFILES.md) · [SHADOWROCKET.md](SHADOWROCKET.md) |

## Security

- Do not distribute uplink/service users meant for secondary plane.
- Firewall on RU: 443/tcp public; SSH key-only.
- Agent API / mTLS: see [FLEET.md](FLEET.md) and doctor output.
