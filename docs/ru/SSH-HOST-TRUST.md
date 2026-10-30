# SSH host-key trust (operator Mac)

## Why `prepare-pack: ssh failed`

Netductor key-only SSH **does not** use `~/.ssh/known_hosts`. Pins live under:

`~/.netductor/secondary/ssh_known_hosts.json`

Manual `ssh root@primary -p52222` can work while deploy fails if this store has no pin for Primary.

## CLI (operator Mac)

```bash
netductor-op ssh-hosts list
netductor-op ssh-hosts show  HOST --port 52222
netductor-op ssh-hosts trust HOST --port 52222 --fingerprint 'SHA256:…'
netductor-op ssh-hosts forget HOST          # full revoke (all ports + legacy)
netductor-op ssh-hosts forget 'HOST|52222'  # one endpoint (refuses if legacy pin remains)
```

`netductor` (node binary) also has `ssh-hosts`; workstation default is **netductor-op**.

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

## Endpoint identity

Pins are stored as `host|port` (e.g. `2.27.118.70|52222`). Legacy bare-host entries still resolve.
Fingerprint comparison is **exact** (case-sensitive Base64). Different ports on the same IP are independent endpoints.

## Recovery (forget)

```bash
# one endpoint only
netductor ssh-hosts forget --kind secondary '2.27.118.70|52222'
# or OpenSSH-style
netductor ssh-hosts forget --kind secondary 2.27.118.70:52222
# all pins for host (legacy + every port)
netductor ssh-hosts forget --kind secondary 2.27.118.70
```

Then re-run `ssh-trust` with a confirmed fingerprint — never auto-accept.

## Mixed legacy + port-aware pins

- New trusts store `host|port` only (not bare host).
- `forget host|port` **refuses** if a legacy bare-host pin still exists — use `forget <host>` to clear all.
- `forget host|port` without legacy removes only that endpoint.
- `forget host` removes legacy and every `host|*`.
- Pin-lost for port B after trusting only port A is **not** raised (ports are independent).
