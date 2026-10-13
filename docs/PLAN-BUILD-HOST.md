# PLAN-BUILD-HOST — dedicated CI VPS (control stays on primary)

Status: **foundation 0.9.361, **prod deploy 0.9.362**** (SSH executor). Agent-pull mode — later if needed.

## Goal

Optional second VPS only for **builds**. Primary keeps:

- job store (`ship-jobs`)
- UI / API / TG / CLI
- git mirrors, publish, releases
- secrets catalog (values injected into remote env for the job only)

Build host runs:

- docker/podman isolate
- compile / test
- returns artifacts to primary

## Modes

| Mode | When | How |
|--|--|--|
| **local** (default) | no build host | current ship worker on primary |
| **ssh** | `build-host` configured + enabled | primary SSH → remote workdir → scp artifacts back |
| **agent** (future) | mTLS poll like secondary | build agent claims queued jobs |

## Config (primary)

File: `/etc/netductor/build-host.json` (or env override)

```json
{
  "enabled": true,
  "ssh_user": "root",
  "ssh_host": "x.x.x.x",
  "ssh_port": 52222,
  "identity_file": "/root/.ssh/id_ed25519",
  "remote_dir": "/var/lib/netductor-build",
  "label": "build-1"
}
```

CLI:

```bash
netductor build-host show
netductor build-host set --host IP [--port 52222] [--user root] [--key PATH] [--enable]
netductor build-host test
netductor build-host clear
netductor build-host disable
```

## Job flow (ssh)

1. Primary claims ship job (vps_slot on primary = “orchestrating”, not local CPU).
2. Sync: git archive / worktree from mirror → scp to `remote_dir/jobs/<id>/src`.
3. Remote: `docker run` with same isolation env + CI secrets as env file (tmp, deleted after).
4. Remote writes `artifacts/`; primary scp’s back to `ship-jobs/<id>/artifacts`.
5. Primary **publish** step still local (releases path on primary disk).

If build host unreachable → job **failed** with clear error (no silent fallback to primary CPU unless `NETDUCTOR_BUILD_HOST_FALLBACK=1`).

## Security

- Key-only SSH; password off on build host after bootstrap (same harden story as secondary).
- Build host **does not** hold fleet VPN private keys or TG token long-term.
- CI secrets: written to remote tmp env file mode 600, removed after step.
- No inbound from internet needed except SSH from primary IP (ufw allow from primary).

## Bootstrap build VPS (operator)

1. Minimal Debian, install docker, git, ca-certificates.
2. Copy primary operator pubkey → `authorized_keys`, harden SSH 52222.
3. `netductor build-host set --host … --enable` on primary.
4. `netductor build-host test`.

Optional later: install `netductor-agent` with `role=build` for heartbeat + cmd queue without SSH.

## Non-goals (now)

- Parallel multi-host load balancing
- Build host as secondary VPN exit
- Replacing Mac external wait (darwin still Mac)


## Prod deploy

```bash
netductor stack apply v0.9.362
netductor build-host deploy --host IP --pass FIRST --key /root/.ssh/id_ed25519 --label build-1
netductor build-host list
netductor build-host watch
```

## 0.9.363 correction

Orchestrator = **operator (Mac)**, same as primary/secondary.
Primary only receives: build-host config, PB/BP tunnel ends, optional heartbeat.

```bash
netductor-op tui  # Wizard → Build-host VPS
netductor-op deploy build-host --host BUILD_IP --pass '…' --primary PRIMARY_IP --key ~/.ssh/netductor_primary
```
