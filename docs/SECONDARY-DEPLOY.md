> Full Blue-Green cutover: [BLUE-GREEN-SECONDARY.md](BLUE-GREEN-SECONDARY.md)

## Blue-Green (v0.9.496+)

```
netductor svc-paths peer add --peer-ip B_IP [--id peer-b]
# apply ForSecondary material on B: bootstrap-secondary
netductor svc-paths peer validate peer-b  # real checks
netductor svc-paths peer ready peer-b
netductor svc-paths peer promote peer-b   # preferred_secondary_ip
# DNS cutover (operator)
netductor svc-paths peer rollback peer-a  # or promote peer-a
netductor svc-paths peer remove peer-b    # only if not active
```

Deploy secondary defaults to Promote=false. Slots: 0=`10.87.10/11`, 1=`20/21`, 2=`22/23`, 3=`24/25`.

**EN** · [RU](ru/SECONDARY-DEPLOY.md)

# Secondary deploy — trust, end-state, failover

Applies from **v0.9.490+** (Stage A) and **v0.9.491+** (Stage B). Path: Mac orchestrates; **no primary→secondary SSH**.

Related: [DEPLOY-MAC.md](DEPLOY-MAC.md) · [SECURITY.md](SECURITY.md) · [MTLS.md](MTLS.md) · [FLEET.md](FLEET.md)

---

## 1. SSH host-key trust (R6 / R16)

Two independent stores:

| Store | Used by | Path / mechanism |
|-------|---------|------------------|
| **Go TOFU** | First password hop (`ProvisionFromCore`, Go SSH) | `secondary` relay known_hosts JSON (`netductor ssh-hosts`) |
| **OpenSSH known_hosts** | Day-2 `ssh`/`scp` via `runSSHOnPort` | `~/.ssh/known_hosts` on the operator Mac |

### Policy (from 0.9.490)

| Situation | Behaviour |
|-----------|-----------|
| First contact (no saved key) | Accept and persist (TOFU / `accept-new`) |
| Key matches saved fingerprint | Continue |
| Key **changed** unexpectedly | **Fail** — no auto-forget, no password retry |
| Operator confirms reinstall / rotation | CLI forget both stores, then re-run deploy (`force_reinstall` optional) |

**Password alone does not wipe trust. Deploy never auto-retries after mismatch (v0.9.493+).**

- Go TOFU: `ForgetSSHHost` only via `ForceReinstall` / `force_reinstall` or CLI `netductor ssh-hosts forget --kind relay <host>`.
- OpenSSH: operator runs `ssh-keygen -R` after verifying the new fingerprint offline.

On mismatch, inspect the new fingerprint offline; only then forget and re-run.

```bash
netductor ssh-hosts list
netductor ssh-hosts forget --kind relay SECONDARY_IP
# then: deploy secondary … (password or key)
```

---

## 2. Deploy sequence (summary)

1. Mac → primary `:52222`: `secondary prepare-pack` (token, mTLS, VPN bundle).
2. Pack JSON: `schema_version`, `min_agent_version`, `agent_id`, material — strict decode (R12/R15).
3. Mac → secondary `:22` (password once) or key: install pubkey, mTLS, `secondary join`.
4. Join rejects unsupported **bundle schema** or agent **older** than `min_agent_version` (R15).
5. Harden → SSH **key-only** on **52222**; password auth off.
6. SP/PS bootstrap: material SCP to secondary `/tmp/nd-svc-material.json` with **`trap EXIT` rm** + mode **600**.
7. Peer-only pings (S→`10.87.10.1`/`10.87.11.1`, P→`10.87.10.2`/`10.87.11.2`); units active; **health timer** active.
8. Wait registry: **this** `AgentID` **online** and `seen=` within **90s** (fresh heartbeat after SP URL).
9. `preferred_secondary_ip` only if registry IP is **IPv4** for that AgentID.

Smoke tests ≠ full HA proof — see §4.

---

## 3. Operator verify (day-2)

```bash
export NETDUCTOR_SSH_PORT=52222
KEY=~/.ssh/netductor_primary   # or your key

ssh -i "$KEY" -p 52222 root@PRIMARY 'netductor secondary status; netductor svc-paths status'
ssh -i "$KEY" -p 52222 root@SECONDARY 'systemctl is-active nd-wss-sp-client nd-wss-ps-server wg-quick@nd-svc-sp wg-quick@nd-svc-ps nd-svc-paths-health.timer; cat /etc/netductor/secrets/secondary_core_url'

# :22 should not accept this key; prefer connection refused
ssh -i "$KEY" -p 22 root@SECONDARY true && echo 'FAIL: :22 still accepts key' || echo 'ok: :22 not usable with key'
```

Expect `secondary_core_url` → `https://10.87.10.1:8789` (SP), not public WAN for agent plane.

---

## Multi-secondary and service plane

Fleet may register **many** VPN secondaries (AgentID, tokens, Reality).

The **service plane** (SP/PS WireGuard + wstunnel) is **one primary ↔ one secondary**:

| Resource | Value |
|----------|--------|
| SP | `10.87.10.1/30` ↔ `10.87.10.2` |
| PS | `10.87.11.1/30` ↔ `10.87.11.2` |
| Ports | WG 51830/51832, WSS 8444/8445 |

Deploying a **different** secondary as HA peer without `--force-new-material` is refused. Force replaces keys and **breaks** the previous secondary’s tunnels.

Re-deploy of the **same** secondary IP **reuses** material (no key churn).

## 4. Failover semantics (v0.9.494+)

| `user_path` | Uplink target |
|-------------|-----------------|
| `vless` | Public primary host |
| `sp` | `10.87.10.1` (SP) |
| `ps` | `10.87.11.1` (PS) |
| `degraded` | No uplink rewrite |

VLESS health = local secondary `:443` listener. Agent URL SP/PS independent. Cycles serialized with flock. Failback uses `StableSeconds` continuous healthy window.

## 4. Failover checklist (R22) — manual

Deploy only proves units + peer ICMP + policy file + timer **active**. Confirm real switch on a lab or maintenance window.

| Step | Action | Expect |
|------|--------|--------|
| 1 | Both SP and PS up | Agent heartbeat fresh; prefers SP |
| 2 | Stop SP transport only (e.g. `nd-wss-sp-client` / SP wstunnel), leave host up | Agent moves to **PS** within policy thresholds (~3 fails) |
| 3 | Heartbeat continues | `secondary status` still online for **this** id; path is PS |
| 4 | Restore SP | Failback after policy window (~45s class) |
| 5 | Stop PS while SP up | Main path stays up |
| 6 | Both SP and PS down | Degraded/offline — not “healthy” |
| 7 | Optional | Client `/sub/<token>` still usable after preferred IP / DNS as designed |

Record: time of fault, time of detect, time of switch, time of failback. Do **not** power off the whole secondary for the SP-only test.

Optional later: `netductor svc-paths selftest` — only after this manual matrix is unambiguous.

---

## 5. Bundle compatibility (R15)

| Field | Meaning |
|-------|---------|
| `version` / pack `schema_version` | Format schema (**1** supported) |
| `min_agent_version` | Secondary binary must be ≥ this SemVer or `join` aborts **before** mutating config |

Producer (primary `prepare-pack` / export) and consumer (`secondary join`) must agree. Upgrade secondary binary before joining a newer pack if min agent moved.

---

## 6. Accepted limitations

| Topic | Stance |
|-------|--------|
| DNS A for `vpn.` host | Manual until a single DNS provider is integrated |
| Static VLESS QR / copied links | May go stale when Reality material rotates; **`/sub/<token>`** is canonical |
| Session API `…/subscription` **410** | Operator API only — not the client subscription channel |
| Auto-retire old secondary | **No** — explicit remove after observation |
| Parallel deploy in one process + global `NETDUCTOR_SSH_PORT` | Avoid concurrent primary/secondary deploys in one op process (R13) |

---

## 7. Related CLI

```text
netductor deploy secondary --primary … --primary-key … --host … --password … --sni …
netductor secondary status          # id, online, ip=, seen=<unix>
netductor secondary prepare-pack    # on primary
netductor ssh-hosts list|forget|clear
netductor svc-paths status          # when available on node
```
