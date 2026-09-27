# Runbook: install & recover (idempotent)

**Canon:** [ARCHITECTURE-FREEZE.md](ARCHITECTURE-FREEZE.md) · recover detail [RECOVER-DRILL.md](RECOVER-DRILL.md)

## Principles
1. **Binaries** always from GitHub Release (`netductor update` / deploy curl) — not from backup.
2. **Data/config** from encrypted backup (`.ndenc`) or secondary recovery API.
3. **Secrets private keys** stay on Mac; VPS keeps pubkeys + service secrets under `/etc/netductor/secrets`.
4. Re-run of the same step is safe: install/harden/domain are designed to be **idempotent**.

## A. Fresh fleet (Mac)

```bash
brew reinstall netductor   # or: netductor-op update
netductor-op tui           # Fleet wizard
# or Web: netductor-op operator serve → /v1/fleet
```

Checklist fields: [DEPLOY-PARITY.md](DEPLOY-PARITY.md).

Same wizard twice on already-hardened hosts: SSH key path + password empty → key-only; components skipped if present.

## B. Update in place (no wipe)

**Primary**
```bash
netductor update                 # or: netductor update 0.9.69
netductor update --component tg
netductor doctor
```

**Secondary**
```bash
# preferred: agent pulls desired_release from primary heartbeat
# manual:
netductor update --component node   # same binary runs agent
systemctl try-restart netductor-secondary-agent sing-box
```

**Mac**
```bash
netductor-op update
# or brew after Formula SHA refresh
```

## C. Recover primary (from secondary)

1. Secondary: `netductor recovery arm` (SSH), note token  
2. New/wiped primary: install node binary from Release  
3. `netductor recover --from-secondary https://SECONDARY:8790 --recovery-token TOKEN --key KEY`  
4. Ensure `NETDUCTOR_OPERATOR_PUBKEY` before harden if recover path injects early  
5. `netductor doctor` · agent heartbeat · VPN test · `https://i…:8443/healthz`

Offline: `netductor recover --key KEY archive.ndenc`

## D. Idempotent checks

| Action | Safe re-run |
|--------|-------------|
| `netductor install` | yes — skips existing units/files |
| `domain set` | yes — overwrites conf |
| `mtls ensure` | yes |
| `update` | yes — replaces binary |
| `recover` | yes if archive valid — **replaces** data |

## E. What this runbook does *not* do
Live wipe of production primary without operator approval.


## Recover automation (0.9.70+)

Without manual steps after wipe:

1. Backup always writes **full baseline** COMPONENTS (dirs…telegram…backup + optional lampac/git/registry).
2. Recover **merges** DefaultComponents even if sidecar was sparse.
3. `netductor-tg` installs as a **real file** at `/usr/local/bin/netductor-tg` (no self-symlink).
4. After restore, **ufw :8789** is opened for secondary public IPs from registry; heartbeat also appends IP.

Drill success criterion: password → recover one-shot → doctor fail=0, bot active, secondary online — **no** manual install/scp/ufw.


## 0.9.71

Pre-restore install may soft-fail `vpn-users` (no secrets yet). Remaining components still install; after tar restore a **second pass** installs api/telegram/backup/vpn-users with secrets present.
