# PLAN: Channel telemetry, alerts, graphs, per-user VPN paths

**Status:** plan only · **Created:** 2026-11-12  
**Goal:** real-time / near-real-time visibility into **all netductor data and control paths**, degradation alerts, history + graphs on **every thin UI**, optional **per-VPN-user** path stats.  
**Constraint:** thin UIs (TG / web / TUI / CLI / op) read **one API + CLI**; no divergent logic (opcatalog / UPDATE contract).  
**Non-goals:** continuous iperf saturating links; hardware deploy in TG.

---

## 1. Channel inventory (all segments)

| ID | Segment | Direction | What “good” means | Primary signals |
|----|---------|-----------|-------------------|-----------------|
| **C1** | Client → Secondary face | Internet → S `:443` (Reality/VLESS) | Accept + low RTT, capacity | TCP connect RTT, optional short bulk, `ss`/sing-box conn count |
| **C2** | Secondary → Primary uplink | S → P public `:443` | Data plane uplink | Existing `uplink_ok` + RTT; **add Mbps estimate** |
| **C3** | Primary face | Internet → P `:443` | Control + public face | TCP RTT from S and/or external; listen health |
| **C4** | Service plane SP | S ↔ P `nd-svc-sp*` (e.g. `10.87.20.0/30`) | WG SP up, low loss | ICMP/TCP to peer VIP, iface rx/tx **rate** |
| **C5** | Service plane PS | S ↔ P `nd-svc-ps*` | WG PS up | same as C4 |
| **C6** | VLESS-svc on SP | S → P `10.87.x.1:9443` | svc inbound up | TCP to AddrP_SP:9443 from S |
| **C7** | Primary egress | P → Internet | Host capacity baseline | DNS RTT, TCP to fixed targets, host default-route iface rate |
| **C8** | Secondary egress | S → Internet | Host capacity on S | agent-reported probes + iface rate |
| **C9** | Edge ↔ cloud | Edge WAN → S/P | edge agent path | existing edge metrics; align series names |
| **C10** | DNS path | clients → blocky → upstream | resolve latency, NXDOMAIN rate | blocky/query metrics if present |
| **C11** | Per-user VPN | user UUID → outbound via S/P | session alive, bytes, last handshake | sing-box / VLESS stats **if exposed**; else aggregate only |

**Yesterday’s 10↓/4↑ Mbps** maps to degradation on **C1 and/or C7/C8** while **C2–C6** can stay “green” on boolean uplink — hence **rates + RTT history** are mandatory, not only ok/fail.

---

## 2. Data model

### 2.1 Sample (one point in time)

```json
{
  "ts": "2026-11-12T06:00:00Z",
  "node": "primary|secondary-id",
  "channels": [
    {
      "id": "C2",
      "peer": "secondary-bb99…",
      "ok": true,
      "rtt_ms": 43.2,
      "loss_pct": 0,
      "rx_bps": 1250000,
      "tx_bps": 400000,
      "note": ""
    }
  ],
  "users": [
    {
      "name": "pavel",
      "uuid_prefix": "a1b2",
      "rx_bytes": 0,
      "tx_bytes": 0,
      "rx_bps": 0,
      "tx_bps": 0,
      "last_handshake_sec": 120,
      "via": "secondary-…"
    }
  ]
}
```

### 2.2 Storage

| Store | Path (indicative) | Retention |
|-------|-------------------|-----------|
| Ring / jsonl samples | `/var/lib/netductor/metrics/channel-history.jsonl` | e.g. 7d @ 15–30s ≈ manageable; downsample to 1–5 min after 24h |
| Latest snapshot | `/var/lib/netductor/metrics/channel-latest.json` | overwrite |
| Alert state | existing notify / alert store | dedupe keys `channel:{id}:{peer}` |

Primary aggregates **own + secondary agent** samples (agent already heartbeats; extend payload or `metrics` cmd).

### 2.3 Measurement methods (ordered by cost)

| Method | Use for | Cadence |
|--------|---------|---------|
| **A. Counter delta** `/proc/net/dev` or WG iface | C4/C5/C7/C8 real throughput | 10–30s |
| **B. TCP dial RTT** | C1–C3, C6 | 15–60s |
| **C. ICMP** (best-effort) | C4/C5 loss hint | 30–60s |
| **D. Short bulk** (256KiB–1MiB, hard timeout 2s) | on-demand + rare scheduled | manual / 15–60 min |
| **E. sing-box stats API** (if enabled) | C11 per-user / per-outbound | 30–60s |

**Never** run sustained iperf on production paths by default.

---

## 3. Collector architecture

```
[timer netductor-channel-sample.timer]
        │
        ▼
 netductor channel sample   ──► channel-history.jsonl + channel-latest.json
        │
        ├── local probes (P)
        └── secondary cmd metrics / heartbeat fields (S)

[evaluateChannelAlerts] ──► notify (TG alerts channel + digest)
```

- Package: extend `internal/channels` (Collect already exists) + thin `internal/channels/sample.go`.  
- CLI:  
  - `netductor channel status [--json]` (enhance)  
  - `netductor channel history [--minutes N] [--id C2] [--json]`  
  - `netductor channel probe [--bw] [--segment …]` (on-demand)  
- Agent: extend metrics/heartbeat with iface rates + optional uplink RTT from S.

---

## 4. Alerts

| Alert key | Condition (defaults, tunable) | Severity |
|-----------|-------------------------------|----------|
| `ch.uplink_down` | C2 ok false for ≥2 samples | P1 |
| `ch.face_down` | C1/C3 TCP fail ≥2 | P1 |
| `ch.sp_down` / `ch.ps_down` | C4/C5 path fail | P1 |
| `ch.rtt_high` | RTT > 3× baseline or > 200ms sustained 5 min | P2 |
| `ch.rate_collapse` | rx+tx bps < 10% of 1h p50 **while** active flows or bulk probe fails | P2 |
| `ch.flap` | ok toggles ≥N in 15 min | P2 |
| `ch.user_stale` | user handshake age > threshold (if C11 available) | P3 |
| `ch.mismatch_surge` | existing mismatch counters spike | P2 (already partial) |

Rules:

- Use **baselines** from history (p50), not only absolute thresholds.  
- Suppress during known `stack apply` / upgrade windows.  
- Batch into existing TG alerts channel (not spam DM).

---

## 5. UI surfaces (all thin UIs)

**Contract:** one backend; surfaces only render.

| Surface | Status snapshot | History / graphs | Alerts |
|---------|-----------------|------------------|--------|
| **CLI** | `channel status` | `channel history` | doctor WARN lines |
| **API** | `GET /api/channel` | `GET /api/channel/history?minutes=` | via existing alerts API if any |
| **Web admin** | Channel panel on Status | sparkline/Chart.js from history | banner |
| **TUI / op** | Status page rows | optional spark ASCII | flag |
| **TG classic** | Status → Channel (enhance tables) | digest lines | alerts chat |
| **TG widgets (dual)** | **info** pane: RTT + Mbps rows + **canvas spark**; **nav**: refresh / probe | spark from last N samples in `chat_data` (≤1024 → **short series only**) | alert text |

opcatalog entries:

- `channel-status` → GET `/api/channel`  
- `channel-history` → GET `/api/channel/history`  
- Surfaces: Web, CLI, TUI, TG (read-only)

### Graphs

| Graph | Series | UI |
|-------|--------|-----|
| RTT by segment | C2, C4, C6, … | all UIs with history |
| Mbps rx/tx by iface/segment | SP, PS, default WAN | web full; TG spark 24–48 pts |
| Per-user top talkers | top 5–10 by bps | web + TG lines; widget table truncated |

TG widget limit: embed **compressed** series in chat_data (e.g. `spk` string `"12,10,9,…"`) already used for CPU — reuse for channel RTT/Mbps on dual **info**.

---

## 6. Per-user VPN monitoring (C11)

### Feasibility

| Source | Feasible? | Notes |
|--------|-----------|-------|
| **sing-box clash/stats / experimental V2** | **Best** if we enable local stats endpoint on localhost | per-outbound / per-user counters |
| **iptables/nft quota** per mark | Medium | needs mark-per-user design |
| **Kernel WG only** | N/A for VLESS users | users are not WG peers on P |
| **Client-side** | Out of band | not control plane |

### Plan

1. **Phase U0:** confirm sing-box version stats API; enable **localhost-only** metrics in generated config (no public expose).  
2. **Phase U1:** map UUID ↔ profile name (existing vpn store); scrape rx/tx; compute bps via delta.  
3. **Phase U2:** API `GET /api/vpn/users/traffic`; TG user card row; web table sort by bps.  
4. **Phase U3:** optional alert `user_heavy` / `user_idle_but_online` — **low priority**, privacy-aware (admin-only).

If stats API unavailable: ship **aggregate** C1–C8 only; document C11 as “best effort / blocked on sing-box feature”.

---

## 7. Phased delivery

| Phase | Deliverable | Depends |
|-------|-------------|---------|
| **T0** | Doc + thresholds defaults + opcatalog stubs | — |
| **T1** | Sampler A+B for C2–C6 on primary; history jsonl; CLI history | T0 |
| **T2** | Secondary agent reports C1/C8 rates + uplink RTT | T1 |
| **T3** | Alerts wired to notify | T1–T2 |
| **T4** | API `/api/channel` + `/api/channel/history` | T1 |
| **T5** | Web graphs + TG dual info sparks + classic Channel table | T4 | **code 0.9.716** |
| **T6** | On-demand `--bw` probe | T1 | **code 0.9.716** |
| **T7** | Per-user C11 if stats available | T4 | **code 0.9.716** |

**Release policy:** each phase shippable; flag `NETDUCTOR_CHANNEL_SAMPLE=1` until soak (mirror widgets gate).

---

## 8. AI / implementer rules

1. **One data path** — UIs must not re-probe ad hoc with divergent logic; call Collect/history.  
2. **No public stats bind** — sing-box metrics only `127.0.0.1`.  
3. **Budget TG chat_data** — series compressed; no full jsonl in widget.  
4. **Dual lifecycle** — channel graphs live on **info** pane; follow `docs/AI-RULES-TG-WIDGETS.md`.  
5. **Do not default-enable heavy bulk probes.**  
6. Prefer **counter rates + RTT** over continuous throughput tests.

---

## 9. Acceptance

Code complete **0.9.716** (sampler/API/UI/probe/users). Checkboxes below need fleet soak.


- [ ] `channel status --json` lists C2–C6 with rtt_ms and rx_bps/tx_bps when traffic present  
- [ ] history ≥ 1h queryable after soak  
- [ ] alert fires on simulated uplink_down  
- [ ] Web + CLI + TG show same latest numbers (± sample skew)  
- [ ] TG dual info shows spark without exceeding zlib/HTML limits  
- [ ] C11 either live or explicitly “unsupported” in status note  

---

## 10. Related

- `internal/channels` (status / path e2e)  
- `internal/metrics` (history.jsonl pattern)  
- `docs/AI-RULES-TG-WIDGETS.md`, `docs/PLAN-TG-DUAL-WIDGET.md`  
- Incident: speed collapse with green uplink → T1 rates required  


## Defaults (starting thresholds)

Agreed 2026-11-12 for lab links (~40–50 ms S↔P, observed collapse ~10↓/4↑ Mbps):

```text
sample_interval: 20s
down_samples: 2
rtt_warn_ms: 150
rtt_crit_ms: 300
rtt_ratio_warn: 3.0
rate_ratio_warn: 0.2
rate_ratio_crit: 0.1
rate_abs_warn_mbps: 15   # only if 1h p50 > 50
rate_abs_crit_mbps: 8
bulk_interval: 45m
bulk_warn_mbps: 20
bulk_crit_mbps: 8
alert_cooldown: 5m
```

Config target: `/etc/netductor/channel-alerts.json` (T3). Constants live in `internal/channels` until then.


---

## 11. Incident capture + rotation (2026-11-13)

**Light sample:** every `collect` when `channel_health` on and `NETDUCTOR_CHANNEL_SAMPLE!=0`.

**On alert (RTT/rate/uplink/face red):** `channels.CaptureIncident` → tar.gz under `/var/lib/netductor/incidents/` → TG document.

**Rotation:**
| Store | Policy |
|-------|--------|
| `channel-history.jsonl` | retain 24h; hard size ≤5 MiB (shrink retain) |
| `incidents/*.tar.gz` | max **20** files, **7 days**, **200 MiB** total |
| `log-exports/` | existing `logs.Rotate` KeepHours |

No public download URL; archive is TG document or path on primary only.
