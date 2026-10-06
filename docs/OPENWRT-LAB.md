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
