# SSH host-key trust (operator Mac)

## Why `prepare-pack: ssh failed`

Netductor key-only SSH **does not** use `~/.ssh/known_hosts`. Pins live under:

`~/.netductor/secondary/ssh_known_hosts.json`

Manual `ssh root@primary -p52222` can work while deploy fails if this store has no pin for Primary.

## Bootstrap (safe)

1. On VPS console (or other out-of-band channel), note the host key fingerprint.
2. On Mac:

```bash
netductor-op ssh-trust 2.27.118.70 --port 52222 --show
# compare fingerprint to console
netductor-op ssh-trust 2.27.118.70 --port 52222 --fingerprint 'SHA256:…'
```

Or with node CLI (on a machine that has `netductor`):

```bash
netductor ssh-hosts show 2.27.118.70 --port 52222
netductor ssh-hosts trust 2.27.118.70 --port 52222 --fingerprint 'SHA256:…'
```

3. Re-run secondary deploy / prepare-pack.

## Rules

- Probe/show is **not** authentication.
- Trust requires explicit `--fingerprint` match.
- Existing pin is never replaced with a different key (forget only via documented recovery).
- Pin lost after `seen` → refuse auto-accept (`ErrPinLost`).
