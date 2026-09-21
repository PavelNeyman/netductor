# Install (Netductor)

Debian-like VPS, root. Target release: **v0.8.16** ([Releases](https://github.com/PavelNeyman/netductor/releases)).

```bash
export NETDUCTOR_VERSION=0.8.16
curl -fsSL https://raw.githubusercontent.com/PavelNeyman/netductor/main/bootstrap.sh | bash
netductor install
# or interactive
netductor tui --mode vps
```

`bootstrap.sh` installs `/usr/local/bin/netductor` from GitHub Release `v$NETDUCTOR_VERSION` (default **0.8.1**).

Components (default all core):

`dirs hardening singbox blocky vpn-users api metrics telegram backup`

```bash
netductor install singbox blocky api
netductor doctor
netductor vpn list
```

Paths: `/etc/netductor`, `/var/lib/netductor`, `/opt/netductor` (data volumes only).

**SSH:** after install, password auth is off — [SSH.md](SSH.md).

**Next:** [RUNBOOK.md](RUNBOOK.md) · [DEPLOY.md](DEPLOY.md) · [FLEET.md](FLEET.md).
