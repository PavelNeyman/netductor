# Deploy from Mac / PC (workstation TUI)

**Baseline: v0.8.57** — operator machine = deployment centre; VPS = control plane after install.

Verified e2e: primary deploy · secondary (no Mac private key on primary) · recover from secondary `:8790` with operator pubkeys from backup **before** harden (no required `NETDUCTOR_OPERATOR_PUBKEY`).

## Requirements on Mac

```bash
brew reinstall netductor   # or Releases darwin binary
netductor version          # ≥ 0.8.57

brew install hudochenkov/sshpass/sshpass   # first password SSH only
```

Also: `ssh`, `ssh-keygen`. Settings: `~/.config/netductor/tui.yaml`.

## SSH model

| Device | After first bootstrap |
|--------|------------------------|
| Primary | Mac pubkey; password **off**; SSH **port 52222** |
| Secondary | Mac pubkey only; primary **does not** store Mac private key; no mesh SSH |
| OpenWrt / MikroTik | Mac pubkey; password off when possible |

Day-2 control: agents → primary **mTLS `:8789`**. Backup offsite: agent **`backup_pull`**, not SCP.

After primary harden, from Mac:

```bash
export NETDUCTOR_SSH_PORT=52222   # optional; deploy primary sets this for later steps
ssh -i ~/.ssh/netductor_primary -p 52222 root@PRIMARY
```

## Control plane ports

| Port | Role |
|------|------|
| `127.0.0.1:8787` | Admin (tunnel: `ssh -L 8787:127.0.0.1:8787 -p 52222 -i KEY root@PRIMARY`) |
| `*:8789` | Agent mTLS |
| `*:52222` | SSH key-only |
| `127.0.0.1:9118` | Lampac (VPN or SSH `-L`) |
| Secondary `:8790` | Recovery API (Bearer `recovery_token`) |

## TUI — full setup order

```bash
netductor tui --mode workstation
# language: L or NETDUCTOR_LANG=ru
```

Use **Setup** menu or **Wizard** tab (same masters).

### 1. Primary VPS

1. Target **Primary VPS**
2. Host / IP, SSH user (`root`), **password** (first login)
3. Generate key? → optional **passphrase** (yes/no → phrase + confirm)
4. TG token + admin id, Reality SNI (e.g. `api.vk.me`)
5. Optional: **Also install Lampac?**
6. Confirm → deploy

Writes `remote_host` / `remote_key` to TUI settings. SSH becomes **52222** + key-only.

Check: `netductor doctor` on primary (via SSH).

### 2. Secondary VPS (RU VPN entry)

Requires primary already in TUI settings.

1. **Secondary VPS** — host, password, SNI
2. Primary key passphrase if any
3. Provision via primary (Mac pubkey → secondary; agent → mTLS)

Save **`recovery_token`** from provision log / secondary `/etc/netductor/secrets/recovery_token`.

### 3. Add-ons (optional)

Setup → **Add-ons** → Lampac on primary over SSH (Docker, localhost only).

### 4. OpenWrt / edge

1. LAN IP (current SSH reachability), device id, arch, `https://PRIMARY:8789`
2. Optional guest Wi‑Fi
3. Optional **Configure LAN/Wi‑Fi/WAN**: dhcp \| static \| pppoe; Wi‑Fi 2.4/5 (empty band inherits the other)
4. Approve edge on primary (TUI / TG / CLI)

Prefer **different LAN subnets** per site; SSID/PSK may be shared.

### 5. Cameras / NVR

Via primary (tunnel). Two-way Tapo audio: plan **go2rtc `tapo://` + WebRTC** (not ONVIF) — after hardware e2e.

### 6. MikroTik

Site / ROS wizard; Mac pubkey harden.

### 7. Day-2

Doctor · VPN users · Edge approve · Admin tunnel · optional git/CI/registry on primary.

## Recover primary from secondary

On a **fresh** VPS (password SSH still on):

```bash
# install binary ≥ 0.8.57, then:
netductor recover --from-secondary http://SECONDARY_IP:8790 \
  --recovery-token '…' \
  --key '…'    # backup decrypt key from /recovery/key or BACKUP_KEY
```

Operator pubkeys are taken **from the backup before harden**. Env pubkey optional.

Then: `ssh -i ~/.ssh/netductor_primary -p 52222 root@PRIMARY`.

## CLI equivalents

```bash
netductor deploy primary --host IP --password '…' --generate-key \
  [--key-passphrase '…'] [--tg-token '…'] [--tg-admin '…'] [--sni api.vk.me] [--with-lampac]

export NETDUCTOR_SSH_PORT=52222
netductor deploy secondary --primary IP --primary-key ~/.ssh/netductor_primary \
  --host RU_IP --password '…' [--sni api.vk.me]

netductor deploy edge --primary IP --primary-key ~/.ssh/netductor_primary \
  --router LAN_IP --password '…' --id site-1 --arch arm64 \
  --server https://PRIMARY:8789 \
  [--configure-net --lan-ip … --wifi-ssid-24 … --wan-proto dhcp|static|pppoe …] \
  [--guest --guest-ssid … --guest-pin …]
```

## See also

[DEPLOY.md](DEPLOY.md) · [FLEET.md](FLEET.md) · [BACKUP.md](BACKUP.md) · [PORTS.md](PORTS.md) · [EDGE-AGENT.md](EDGE-AGENT.md)
