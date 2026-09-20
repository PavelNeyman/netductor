# Deploy from Mac / PC (workstation TUI)

Baseline: **v0.8.2**. Operator machine runs TUI as the **deployment centre**; VPS remains the control plane after install.

## Requirements on Mac

- `netductor` binary for darwin ([Releases](https://github.com/PavelNeyman/netductor/releases))
- `ssh`, `scp`, `ssh-keygen`, `curl`
- First login to a **password** SSH host: `sshpass` (`brew install hudochenkov/sshpass/sshpass` or equivalent)
- After primary deploy: key-only SSH via `~/.ssh/netductor_primary` (saved in TUI settings)

## TUI flow

```bash
netductor tui --mode workstation
```

**Setup wizard:**

1. **Primary VPS** — host, root password, generate/path SSH key, TG token + admin id, SNI → downloads linux binary on VPS, secrets, `install`, SNI, fleet bootstrap; writes `~/.config/netductor/tui.yaml`.
2. **Secondary VPS** — RU host + password; uses primary key from settings → `fleet provision-secondary` on primary.
3. **OpenWrt** — LAN IP, device id, agent arch; token from primary, agent download, `edge provision`.
4. **Cameras / NVR** — leases / add / probe / record via primary.
5. Day-2: remote settings; Doctor, VPN, Fleet, Edge approve.

## CLI

```bash
netductor deploy primary --host IP --password '…' --generate-key --tg-token '…' --tg-admin '…'
netductor deploy secondary --primary IP --primary-key ~/.ssh/netductor_primary --host RU_IP --password '…'
netductor deploy edge --primary IP --primary-key ~/.ssh/netductor_primary --router 192.168.1.1 --id cudy-home-1 --arch arm64
```

See also [DEPLOY.md](DEPLOY.md) · [FLEET.md](FLEET.md) · [EDGE-AGENT.md](EDGE-AGENT.md).
