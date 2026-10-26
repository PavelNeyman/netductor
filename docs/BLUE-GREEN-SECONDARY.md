# Blue-Green Secondary (R5 acceptance)

**Code baseline:** v0.9.501+

> This document is an **operator acceptance procedure**, not a production certificate.
> Preflight prints state only; a zero exit does **not** mean cutover PASS.
> Live traffic, fault-injection recovery, and dual-node isolation must be signed off on the lab/prod VPS.  
**Goal:** Run Secondary B next to A, validate, promote, DNS cutover, rollback — without breaking A’s service plane.

## Preconditions

- Primary on v0.9.499+ with `svc-paths` and inventory APIs.
- Secondary A already HA (slot 0 typical): SP/PS up, agent heartbeat OK.
- Secondary B: fresh VPS, SSH from Mac, same version binary path as deploy.
- DNS TTL for VPN hostname lowered (60–300s) **before** promote day.
- Operator understands: **register ≠ peer add ≠ promote ≠ DNS**.

## Phases

### 0. Register candidate (optional)

```bash
# Operator / Mac inventory (no network change)
curl -sS -H "Authorization: Bearer $OP_TOKEN" -H 'Content-Type: application/json' \
  -d '{"name":"ru-b","public_ip":"B_PUBLIC_IP"}' http://127.0.0.1:PORT/v1/secondaries

# Or on Primary
netductor inventory add --name ru-b --ip B_PUBLIC_IP
```

Telegram: Fleet → Secondaries → Register (`name IP`).

### 1. Add service-plane peer on Primary (does not promote)

```bash
# On Primary
netductor svc-paths peer add --peer-ip B_PUBLIC_IP --id peer-b
# stdout JSON = ForSecondary material — copy to B
```

Checks after add:

- `peer list` shows `peer-b` state `prepared`, **new slot** (not 0 if A holds 0).
- A still pings: `ping 10.87.10.2` (or A’s AddrS_SP).
- Units for A still active (`wg-quick@nd-svc-sp`, `nd-wss-sp-server`, …).
- Legacy `material.json` still matches **A** if A was first peer.

If add interrupted: `netductor svc-paths peer recover peer-b`.

### 2. Bootstrap B

Apply material on B (`bootstrap-secondary` / deploy path).  
Deploy from Mac with **Promote=false** (default).

On B:

```bash
netductor svc-paths peer validate peer-b   # or without id uses local material
systemctl is-active wg-quick@nd-svc-sp1   # if slot 1
# agent heartbeat visible on Primary
```

### 3. Ready

On B, validate only (local material and units):

```bash
netductor svc-paths peer validate peer-b
```

On Primary, after the client traffic test:

```bash
netductor svc-paths peer validate peer-b
netductor svc-paths peer ready peer-b
netductor svc-paths peer promote peer-b
```

`ready` and `promote` need the Primary peer registry. They are not run on B.

`--force-ready` is ops escape hatch only.

### 4. Client path test **before** promote

- Point a test client at **B’s IP** (not DNS) with B’s Reality params from B.
- Confirm TCP/UDP user traffic.
- Optional: break SP on B, confirm PS failover on B only; A unchanged.

### 5. Promote (preferred_secondary_ip)

```bash
netductor svc-paths peer promote peer-b
# writes preferred_secondary_ip + active_peer_id
```

Does **not** change DNS by itself.

### 6. DNS cutover

Update A record VPN hostname → B.  
Watch new sessions; old TCP may stick until reconnect.

### 7. Observe window

Keep A up (plane + agent). Monitor errors, heartbeats both nodes.

### 8. Rollback

```bash
netductor svc-paths peer promote peer-a   # or: peer rollback peer-a
# DNS back to A if needed
```

### 9. Remove old peer (only when not active)

```bash
netductor svc-paths peer remove peer-a
```

## Acceptance checklist (sign-off)

| # | Check | Pass |
|---|--------|------|
| 1 | `peer add` B does not stop A units / does not rewrite A WSS unit restrict-to | ☐ |
| 2 | A and B different slots and WSS_SP ports | ☐ |
| 3 | B validate OK (units, peer ping, :443) | ☐ |
| 4 | promote refused while `prepared` | ☐ |
| 5 | promote B updates preferred; clientcfg prefers B | ☐ |
| 6 | Test client via B IP works before DNS | ☐ |
| 7 | DNS cutover; new clients reach B | ☐ |
| 8 | Rollback to A restores preferred; traffic recoverable | ☐ |
| 9 | `peer remove` non-active; after reboot removed units stay down | ☐ |
| 10 | Corrupt material.json does **not** silent-regen keys | ☐ |

## Explicit non-goals of R5 automation

- Auto DNS providers API.
- Auto promote from Telegram.
- Shared Reality key between A and B (may differ — align `/sub` after promote).

## peer-ops.log (audit, not recovery)

`/etc/netductor/svc-paths/peer-ops.log` records add/ready/promote/remove/recover **for operators**.

- **Source of truth** for recovery: `peers.json`, `peers/<id>/op.json`, `peers/<id>/material.json`, preferred/active files.
- Journal write failure does **not** roll back a successful plane mutation (promote/add index). stderr warns: `peer-ops journal … plane op already committed`.
- Do not treat the log as complete audit: lines may be missing after disk errors.

## DeploySecondary and multi-peer

From **v0.9.510**, `netductor deploy secondary` / operator deploy calls **`svc-paths peer add`** on primary (not legacy `bootstrap-primary` alone). Material uses a free slot; verification on B is slot-aware (`iface_sp`/`addr_p_sp` from material).

- Default **Promote=false** — does not write preferred_secondary_ip; use `peer ready` / `peer promote` after traffic test.
- First secondary still may use slot 0; a second deploy allocates the next free slot without rewriting A’s legacy material.

## Promote flag on deploy

- Default **Promote=false**: provision only; operator runs `peer validate` / `ready` / `promote` after traffic proof.
- **Promote=true**: runs `peer validate` then `ready` (no `--force-ready`) then `promote` on primary. Still does **not** replace live user-traffic R5 or DNS cutover.
- Emergency force-ready remains CLI-only: `peer ready <id> --force-ready`.

## Subscription profiles (v0.9.535)

`GET /sub/{token}` stays the live URL. `sub_profile=secondary` returns every recently seen Secondary URI, not only the preferred one.

- Shared hostname from `NETDUCTOR_VPN_HOST` or `/etc/netductor/vpn_hostname`.
- Each URI keeps that Secondary's `pbk`/`sid`/`sni` and a unique `#nd-secondary-<ip>` label.
- Stale or never-seen (`LastSeen` zero or older than 3 minutes) devices are omitted.
- `primary` is still core only. `both` is all Secondary URIs plus core.
- DNS cutover is manual. `peer promote` does not change DNS.

Two different removes:

| Command | What it drops |
|--|--|
| `netductor secondary remove <agent-id>` | VPN device in `secondary/devices.json` — this is what removes A from `/sub` |
| `netductor svc-paths peer remove <peer-id>` | Service-plane peer/slot on primary. Does not by itself drop the subscription URI |

Do not remove A until B traffic and DNS cutover are confirmed. `peer remove` refuses the active peer and no longer marks `removed` if unit/file cleanup failed.

## Migration runbook (v0.9.539)

Separate Reality profiles. DNS does not switch a profile. `prepared` is not traffic-ready. Promote is not automatic.

1. Provision B (`Promote=false`). Primary SSH port comes from `PrimarySSHPort` (default 52222).
2. Infrastructure checks: units, ping, fresh heartbeat, non-empty PBK/SID, SingBoxOK, UplinkOK.
3. External client traffic test against B's public IP with B's profile. Not part of deploy success.
4. `peer ready` then `peer promote` only after that traffic test.
5. Users pick the B profile from `/sub/{token}`. Both A and B are listed while both are fresh.
6. Optional manual DNS cutover of the shared hostname. Not a substitute for the B profile.
7. Observation window. Keep A up for rollback.
8. Remove A only after sign-off: stop A services, `secondary remove <agent-id>` (drops A from `/sub`), then `peer remove <peer-id>`.
9. Rollback before the window ends: DNS back to A, promote A. Do not remove A first.

`peer add` of a removed id fails. Use a new id or recover. Empty SID/PBK is omitted from `/sub`.

## 0.9.540

- Deploy never promotes. After the client traffic test, run `peer ready` / `peer promote` yourself.
- `--primary-ssh-port` and `--peer-id` flow CLI → SecondarySpec → DeploySecondary.
- `/api/secondary/links` skips empty SID, same as `/sub`.
- After `peer remove`, reuse of the same IP needs `--peer-id` (a new id) or recover. Default `peer-<ip>` will be refused.

## 0.9.545

- On B, `peer validate` uses local `material.json`. An id is optional and must match if set. On primary, `peer validate <id>` uses `peers/<id>`.
- `peer remove` also deletes the peer's UFW rules and `peers/<id>` material. It does not stop services on the old Secondary itself.
- `ready` is infra-only. It does not prove client Reality traffic. Promote only after that test.
- Shadowrocket helper keeps both profiles when they share a hostname but differ in Reality keys.


## 0.9.546

- UFW cleanup errors if status cannot be read. Inactive UFW still has its peer rules deleted.
- On B, a requested peer id must match material PeerID exactly, including a non-empty id.
- `peer remove` writes `removing` before deleting material, so a failed reload can be retried.
- Ready and promote run on Primary, not on B.
