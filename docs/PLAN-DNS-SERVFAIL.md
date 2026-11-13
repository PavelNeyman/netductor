# PLAN: DNS SERVFAIL spike (sing-box journal)

**Date:** 2026-11-13  
**Trigger:** TG alert `⚠️ DNS SERVFAIL spike: N in 15m (sing-box journal)`  
**Detector:** `cmd/netductor/collect.go` — `countJournalMatches("sing-box", 15, "SERVFAIL")`, threshold **≥ 40 / 15m**, key `dns:servfail`.

This is **not** the same as blocky “ads NXDOMAIN”. SERVFAIL = resolver failed (upstream timeout, refused, broken chain, or misconfigured local DNS path).

---

## 1. What the alert means

| Item | Meaning |
|------|---------|
| Source | Lines in `journalctl -u sing-box` containing `SERVFAIL` |
| Where it hurts | DNS answers for **user traffic** that go through sing-box DNS (hijack-dns / embedded DNS) |
| Typical user symptom | Slow page opens, apps “no network”, intermittent fails — can **look like** low VPN speed |
| Not implied | SP/wstunnel down, or primary egress Mbps collapse |

Correlate with channel metrics: if RTT/uplink OK but UX bad → DNS first.

---

## 2. Immediate triage (primary + secondary)

Run on the node that fired the alert (usually primary; if secondary has its own sing-box DNS, check both).

```bash
# 1) volume + samples
journalctl -u sing-box --since '20 min ago' --no-pager | grep -c SERVFAIL || true
journalctl -u sing-box --since '20 min ago' --no-pager | grep SERVFAIL | tail -40

# 2) local resolvers
systemctl is-active blocky sing-box
ss -ulnp | grep -E ':53|:853|:7878' || true
cat /etc/netductor/singbox-config-variant 2>/dev/null

# 3) blocky health (if used)
dig @127.0.0.1 example.com +time=2 +tries=1
dig @127.0.0.1 doubleclick.net +short   # often NXDOMAIN if ads OK
journalctl -u blocky -n 40 --no-pager

# 4) upstream from host (bypass stack)
dig @9.9.9.9 example.com +time=2
dig @1.1.1.1 example.com +time=2

# 5) sing-box DNS shape (1.12 type+server vs legacy address)
python3 - <<'PY'
import json
c=json.load(open("/usr/local/etc/sing-box/config.json"))
print("dns:", json.dumps(c.get("dns"), indent=2)[:2000])
print("route.rules dns-ish:", [r for r in (c.get("route") or {}).get("rules") or [] if "dns" in str(r).lower()][:10])
PY
```

Record: **which domains**, **which outbound DNS tag**, **primary vs secondary**, concurrent **channel status**.

---

## 3. Likely causes (priority order)

| # | Cause | Evidence | Direction |
|---|--------|----------|-----------|
| A | **Upstream DoH/DoT/UDP flaky** (9.9.9.9, Quad9 TLS, etc.) | Host `dig @9.9.9.9` timeouts; journal SERVFAIL on many names | Retry/failover DNS servers; shorter timeout; secondary upstream |
| B | **Blocky down / not on :53 / bypass** | `systemctl` inactive; dig @127.0.0.1 fail; traffic skips blocky | Fix unit + ensure sing-box DNS points at blocky when variant expects it |
| C | **sing-box 1.12 DNS migration** (`domain_resolver` / server form) | WARN on `sing-box check`; broken server entries | Align generated config with pin 1.12 (already partially done) |
| D | **Uplink-dependent DNS** (resolve via detour that is busy/down) | SERVFAIL spikes with uplink RTT/mux EOF | Decouple critical DNS from sick uplink (local/bootstrap resolvers) |
| E | **Rate / concurrency** | Spike only under load | Cache, fewer parallel upstreams, avoid stampeding same upstream |
| F | **Noise: blocked/ads path logged as SERVFAIL** | Only ad/tracker names | Raise threshold or filter journal pattern (don’t treat as outage) |

Do **not** assume “VPN tunnel protocol” until A–C cleared.

---

## 4. Product / code plan

### P0 — ops clarity (no behavior change)

1. Enrich alert text: node role, top N domains from journal (sanitized), `blocky` active yes/no.  
2. Attach **light** incident snippet (last 50 SERVFAIL lines) via existing incident path **with long cooldown** (e.g. 45m) — avoid tar flood.  
3. Document in doctor: `dns_servfail_15m` count + blocky/:53 checks.

### P1 — resilience

1. **Bootstrap DNS** always reachable without user uplink (local blocky or fixed IPs).  
2. Dual upstream in generated DNS (e.g. Quad9 + Cloudflare) with strategy already supported by sing-box.  
3. Ensure secondary DNS path does not hard-depend on broken mux for *resolver* queries.  
4. Review `hijack-dns` vs system resolvers so host collect/dig still work.

### P2 — alert quality

1. Threshold: keep ≥40/15m or make configurable `NETDUCTOR_DNS_SERVFAIL_THRESHOLD`.  
2. Optional: count only lines with `response_code=SERVFAIL` / exclude known ad lists if logged that way.  
3. Clear alert only after **two** consecutive collect windows below threshold (anti-flap), same idea as channel RTT.

### P3 — UX

1. Channel/DNS row in TG status: SERVFAIL rate 15m.  
2. Link from alert to `netductor doctor` DNS section.

---

## 5. Relation to speed complaints

- SERVFAIL → retries → **latency and “slow internet”** without low iperf.  
- After spike: check whether user reports match DNS (pages fail) vs pure throughput (speedtest number).  
- Fixing DNS is **orthogonal** to S→P uplink A/B but should be done **before** blaming Reality/mux alone.

---

## 6. Out of scope (this plan)

- Replacing S→P protocol (see separate uplink A/B plan).  
- Changing blocky blocklists as “fix” for SERVFAIL (NXDOMAIN ≠ SERVFAIL).  
- Public recursive open resolver on WAN.

---

## 7. Acceptance

| Done when | |
|-----------|--|
| Triage commands captured once per incident class | |
| Alert actionable (role + sample domains) | |
| No sustained ≥40 SERVFAIL/15m under normal load | |
| Doctor surfaces DNS path health | |
| Speed regression re-checked after DNS stable | |

---

## 8. First code touch (when implementing)

- `cmd/netductor/collect.go` — enrich `dns:servfail`  
- `internal/vpn/*` DNS builders — bootstrap + dual upstream if gaps  
- `doctor` / channel status — expose count  
- Tests: journal counter threshold boundary; no incident spam

**Do not** ship uplink protocol migration in the same change as DNS alert tuning.
