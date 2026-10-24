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

On the host that can see B’s plane (usually B or Primary after tunnels up):

```bash
netductor svc-paths peer validate peer-b
netductor svc-paths peer ready peer-b
```

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

