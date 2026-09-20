# Deploy from Mac / PC (workstation TUI)

Baseline: **v0.8.5**. Operator machine runs TUI as the **deployment centre**; VPS remains the control plane after install.

## Requirements on Mac

- `netductor` binary for darwin ([Releases](https://github.com/PavelNeyman/netductor/releases) or `brew install netductor` / `brew install --HEAD netductor`)
- `ssh`, `scp`, `ssh-keygen`, `curl`
- First login to a **password** SSH host: `sshpass` (`brew install hudochenkov/sshpass/sshpass` or equivalent)
- After primary deploy: key-only SSH via `~/.ssh/netductor_primary` (path saved in TUI settings)

## SSH keys (operator only)

| Device | After first password bootstrap |
|--------|--------------------------------|
| Primary | Mac `netductor_primary.pub` in `authorized_keys`; password auth **off** |
| Secondary | **Mac** pubkey installed (`--operator-pubkey`); password **off**. Primary does **not** need a mesh SSH key to secondary |
| OpenWrt | Mac pubkey + dropbear/OpenSSH password **off** when possible |
| MikroTik | Mac pubkey via `/user ssh-keys`; password login best-effort off |

Ongoing control is **not** device↔device SSH. Secondary/edge agents talk to primary over **HTTP(S)** (see transport below).

## Control traffic (agent → primary)

| Plane | Bind / URL | Security |
|-------|------------|----------|
| Admin API | `127.0.0.1:8787` | Local only. From Mac: `ssh -L 8787:127.0.0.1:8787 root@PRIMARY` |
| Secondary agent | **mTLS `:8789` only** | Auto certs; UFW from secondary IP; plain 8788 emergency-only |
| Edge agent | **mTLS `:8789`** | Client certs auto-installed; works if site VPN is down |

Workstation edge deploy may still seed `http://PRIMARY:8787` for first enroll if the public enroll endpoint is open. After VPN is up, prefer in-tunnel or HTTPS URL.

## TUI flow

```bash
netductor tui --mode workstation
```

**Setup wizard:**

1. **Primary VPS** — host, root password, generate/path SSH key, TG token + admin id, SNI → binary on VPS, secrets, `install` (SSH harden, **no** extra mesh key if Mac pubkey already present), SNI, fleet bootstrap; writes `~/.config/netductor/tui.yaml`.
2. **Secondary VPS** — RU host + password; Mac pubkey passed as `--operator-pubkey` → provision on primary → agent polls primary (HTTP/mTLS).
3. **OpenWrt** — LAN IP, device id, arch; agent + Mac pubkey harden; enroll/approve on primary.
4. **Cameras / NVR** — leases / add / probe / record via primary API (through tunnel or allowed path).
5. **MikroTik** — site RSC push + optional SSH harden with Mac pubkey.
6. Day-2: Doctor, VPN, Fleet, Edge approve.

## CLI

```bash
netductor deploy primary --host IP --password '…' --generate-key --tg-token '…' --tg-admin '…'
netductor deploy secondary --primary IP --primary-key ~/.ssh/netductor_primary --host RU_IP --password '…'
netductor deploy edge --primary IP --primary-key ~/.ssh/netductor_primary --router 192.168.1.1 --id cudy-home-1 --arch arm64
```

See also [DEPLOY.md](DEPLOY.md) · [FLEET.md](FLEET.md) · [EDGE-AGENT.md](EDGE-AGENT.md) · [ARCHITECTURE.md](ARCHITECTURE.md).
