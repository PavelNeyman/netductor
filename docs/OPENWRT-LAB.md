# OpenWrt lab & stock deploy contract

## Product goal (non‑negotiable)

```text
Factory / stock OpenWrt  →  one netductor edge deploy  →  fully configured site router
```

- **No** manual UCI on the device before deploy.
- **No** “set LAN to DHCP client” for lab or prod.
- Same path as a real Cudy after reset: empty/root password, LAN `192.168.1.1/24`, SSH `:22`, WAN gets uplink if plugged.

Deploy must install agent, enroll, apply template (LAN/WAN/Wi‑Fi/guest/firewall/VPN as designed), harden SSH, leave a usable router.

## Stock assumptions (what deploy may rely on)

| Item | Factory typical |
|--|--|
| SSH | `root@192.168.1.1`, password empty or vendor default |
| LAN | static `192.168.1.1/24`, DHCP server on |
| WAN | dhcp client on `wan` when cable/uplink present |
| Wireless | `wifi-device` radio* present on real hw; often **absent** in pure VM |
| Flash/arch | real: mipsle/arm; lab VM: amd64/arm64 agent asset |

Agent already: empty password SSH, stdin file put (no SFTP), radio detect via UCI (not Cudy-only `radio0`).

## Lab topology — emulate a PC on the LAN, not “OWRT as laptop”

Wrong (do not do this for netductor lab):

```text
Mac NAT  →  OpenWrt forced to DHCP client on lan
```

That changes the device away from stock and trains the wrong muscle memory.

Right:

```text
                    ┌──────────── OpenWrt VM (STOCK) ────────────┐
  Internet/VPN      │  eth0 WAN: dhcp client → reaches primary   │
  (Mac or uplink) ──┤  eth1 LAN: 192.168.1.1/24, DHCP server    │
                    └──────────────────┬─────────────────────────┘
                                       │ host-only / socket_vmnet shared
                    Mac as LAN client ─┘  e.g. 192.168.1.2
                    deploy: root@192.168.1.1
```

- **WAN NIC**: NAT or shared so the **router** can reach primary `:8789` (enroll/heartbeat) without you editing UCI.
- **LAN NIC**: host-only (or shared subnet `192.168.1.0/24` where OWRT keeps `.1` and Mac is `.2`).
- Operator runs deploy against **`192.168.1.1`** exactly like after a factory reset on the desk.

If the hypervisor only offers one NIC: still keep OWRT LAN stock; put the Mac on that L2 (bridge/shared) so it can SSH to `192.168.1.1`, and ensure OWRT WAN path to primary exists (second NIC preferred).

### Lima / vfkit (no UTM)

Prefer tools that can attach **two** virtio-nets:

1. NIC0 → NAT (WAN)
2. NIC1 → host-reachable network where you assign Mac `192.168.1.2/24` and leave guest LAN default

`vzNAT` alone makes the **guest** a DHCP client on one interface — fine for a Linux server VM, **wrong** as the only model for a stock router lab. Use vzNAT/NAT only on the **WAN** side of a dual-NIC OpenWrt VM.

`socket_vmnet` shared/host: good for the LAN leg so the Mac talks to `192.168.1.1`.

## Deploy checklist (stock → ready)

Operator (Mac / op):

```bash
# Mac has route to 192.168.1.1 (LAN leg)
ping -c1 192.168.1.1
netductor-op edge deploy --host 192.168.1.1 --arch amd64   # or arm64 / mipsle
# approve enroll on primary if required
```

Expect without hand-editing the router:

1. Agent binary + config + optional mTLS  
2. Enroll / approve  
3. Template / bootstrap apply (network, wireless, guest policy as template defines)  
4. Harden (key-only SSH, etc.)  
5. Heartbeat to primary  

Gaps to close in code/product if still manual today (track in OPEN_ITEMS): any step that still requires LuCI or SSH UCI after deploy is a bug relative to this contract.

## VM vs real hardware

| | Stock deploy path | Guest SSID / RF |
|--|--|--|
| Dual-NIC OpenWrt VM | Yes (agent, template, harden) | UCI only; no air |
| Real Cudy/OpenWrt | Yes (mipsle asset) | Full e2e |

## Radio

On hardware, `detectWifiDevices()` + guest apply; no Cudy-only hardcode.  
On VM without `wifi-device`, guest UCI may reference `radio0` fallback — harmless until real radios exist.


## Operator deploy contract (hybrid)

Shared backend: `internal/edge` (Facts/Plan/Selection) + `internal/deploy` (SSH probe/apply).  
Thin UIs: op CLI/TUI and Web call the same shapes (`PlanResponse` / `/api/edge/plan`). **No Telegram deploy.**

### Factory (device may have no internet)

```bash
# On LAN host that can reach root@192.168.1.1
netductor-op deploy edge --router 192.168.1.1 --id SITE \
  --preset sbc-lab|travel-router \
  --configure-net --guest \
  --dry-run --json          # same JSON as POST /api/edge/plan

netductor-op deploy edge ...   # real provision over SSH
```

- Probe → plan → modules over **SSH only**.
- Empty `--preset` → auto from facts (single NIC → `sbc-lab`).
- Offline: `deploy offline-prep` caches agents, token, mTLS, **templates** under `~/.cache/netductor/agents/templates/`.

### Day-2 (agent online)

- Primary: `GET /api/edge/card?id=`, `POST /api/edge/plan`, `POST /api/edge/facts`.
- Agent heartbeat carries `facts` for the card.

### MikroTik

**Not** in this OpenWrt UCI path. Separate track (ROS SSH/API, shared site labels only). Do not force OpenWrt presets onto RouterOS.

