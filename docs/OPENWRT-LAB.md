# OpenWrt lab (deploy e2e without only Cudy hardware)

## Goal

Exercise **edge agent** paths: enroll, heartbeat, template apply, guest Wi‑Fi UCI, remote cmds — not full RF certification.

## Mac vs VPS

| Host | Pros | Cons |
|--|--|--|
| **Mac (QEMU/UTM)** | Local, fast iterate, snapshot | ARM/x86 OpenWrt image; **not** mipsle Cudy; no real radios |
| **VPS (KVM/QEMU)** | Always on, close to primary network if same DC | Same arch limits; need nested virt or full VM |
| **Real Cudy/OpenWrt** | True mipsle + radios | Required for final RF / guest SSID air test |

**Recommendation:** day-to-day on **Mac UTM/QEMU x86_64 or aarch64 OpenWrt**; keep **one real router** for radio/guest e2e. VPS only if you want a shared lab next to primary.

## Quick Mac path (UTM)

1. Download OpenWrt **combined-ext4** or **generic** image for your arch (x86_64 or armsr).
2. UTM → new VM, virtio NIC bridged or shared network so it reaches primary `:8789`.
3. First boot: set password, enable SSH, install nothing heavy.
4. From operator:

```bash
netductor-op edge deploy --host <vm-ip> --arch amd64   # or arm64
# agent enroll → approve on primary
netductor edge list
```

5. Template / guest apply from TG or CLI; inspect on VM:

```bash
uci show wireless
uci show network | grep guest
logread -e netductor
```

Radios in pure VM are often **empty** → agent falls back to `radio0` UCI names (harmless) or set:

```bash
# on agent host
export NETDUCTOR_GUEST_RADIOS=radio0,radio1
```

## VPS path

Same as Mac if VPS has KVM: run OpenWrt VM, attach second vNIC to lab VLAN or wireguard to primary. Nested QEMU without KVM is painful — prefer Mac or bare metal mini-PC.

## Scenarios checklist

- [ ] Enroll + approve
- [ ] Heartbeat / version
- [ ] Template apply (network baseline)
- [ ] Guest apply (UCI sections on detected radios)
- [ ] Guest disable / cleanup
- [ ] Agent update with SHA
- [ ] Recovery / offline path (optional)

## Radio detect (agent)

`detectWifiDevices()` reads `uci show wireless` for `wifi-device`, band from `band` or `hwmode`. Guest APs bind to detected radios (prefer one 2g + one 5g). Override `NETDUCTOR_GUEST_RADIOS`.


## Alternatives to UTM (Apple Virtualization)

| Tool | Backend | Network that usually works | Notes |
|--|--|--|--|
| **Lima + vmType vz + vzNAT** | Virtualization.framework | Guest gets IP on host-visible subnet (often `192.168.64.x`); host can SSH to guest IP | **Best first try** for edge agent lab; no `socket_vmnet` required |
| **Lima + socket_vmnet shared** | QEMU/vz + vmnet | Host↔guest on `192.168.105.0/24` | Needs `brew install socket_vmnet` + `limactl sudoers` |
| **Lima + socket_vmnet bridged** | vmnet bridged to `en0` | Guest on LAN DHCP | Fragile on Wi‑Fi; needs root launchd; Apple restricts true bridge entitlements |
| **vfkit** | Virtualization.framework CLI | Default **NAT** (`--device virtio-net,nat`); IP in `/var/db/dhcpd_leases` | Bridged needs vmnet-helper / restricted entitlements |
| **Tart** | VZ | softnet / socket_vmnet plugins | Aimed at macOS/Linux images; OpenWrt is DIY |
| **UTM** | QEMU or Apple VZ | Bridge often flaky | You already saw glitches |

### Recommended path (Lima vzNAT)

```bash
brew install lima
# OpenWrt is not a stock Lima template — use a plain Linux first to prove network:
limactl start --name=netlab --vm-type=vz --network=vzNAT template://ubuntu-lts
limactl shell netlab -- ip -4 addr
# From host: ssh to the vzNAT guest IP (lima shows it / ifconfig bridge)
```

For **OpenWrt disk image**: convert to raw, boot with vfkit/Lima custom YAML (`firmware`, `disk`, `vmType: vz`, `networks: [{vzNAT: true}]`).  
In guest set **one** NIC to DHCP client (not OpenWrt “LAN static 192.168.1.1” fighting the hypervisor DHCP):

```bash
uci set network.lan.proto='dhcp'
uci delete network.lan.ipaddr
uci delete network.lan.netmask
uci commit network
/etc/init.d/network restart
```

### Why DHCP/static “does not work”

1. **OpenWrt default** assumes it **is** the router (static LAN). Hypervisor NAT/vzNAT expects a **DHCP client** → conflict.
2. **Bridged on Wi‑Fi** is unreliable (Apple/vmnet + AP isolation). Prefer **Ethernet** or stick to **vzNAT/shared**.
3. **True bridge** needs `com.apple.vm.networking` entitlement — most open tools only do **NAT/shared**.
4. Static IP without matching host gateway/DNS on the **same** vmnet subnet → no internet.

### Practical lab topology for netductor

```
Mac --vzNAT--> OpenWrt-VM (dhcp client, SSH)
Mac --VPN/SSH--> primary VPS
edge deploy from Mac to VM IP on vzNAT
```

Real radio/guest SSID still needs physical OpenWrt.
