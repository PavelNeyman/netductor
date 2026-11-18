# G115 integer-overflow audit (ai-tools review)

| | |
|--|--|
| **Date** | 2026-11-18 |
| **Source report** | `aitools-report-netductor-v1121` (gosec Issues, rule G115) |
| **Baseline SHA** | post-N1/N2/N3 (`0ee2870` tip at start of pass) |
| **Count** | 43 findings / 16 files |

G115 = “integer overflow conversion” (narrowing). Not all are real bugs: many are **false positives** on Linux UIDs or protocol-bounded sizes. Each site is classified below. Fixes go in the same commit as this journal when marked **fixed**.

## Classification legend

| Class | Meaning |
|--|--|
| **FP-uid** | Compare `Stat_t.Uid` (uint32) to `os.Geteuid()` (int). Kernel UID always fits uint32; risk is only theoretical negative euid. |
| **safe-bound** | Value already constrained by prior check or constant max. |
| **real** | Conversion without a bound that the attacker/peer can influence. |
| **protocol** | Required by on-wire format (CCM, KLAP, PKCS7); sizes fixed by protocol. |

## Non-tapo (actionable)

| # | Location | Class | Decision |
|--|--|--|--|
| 1–17 | `*Uid != uint32(os.Geteuid())` in agent recovery, api_secondary, firewall, mtls/revoke, secondary/store, svcpaths peer + bootstrap journal | FP-uid | **Fixed**: compare in `int64` space: `int64(uid) != int64(os.Geteuid())` — no narrowing, fail-closed if euid ever negative. |
| 18 | `firewall.go` host walk `uint32(start+offset)` | safe-bound | ones≥16 ⇒ count≤65536. **Fixed**: explicit `hostU32 > 0xffffffff` guard before cast. |
| 19–21 | `install/backup.go` `PutUint32(len(ct))` + max check with `gcm.Overhead()` | safe-bound | chunk ≤ 1MiB + tag. **Fixed**: `ndg2PutChunkLen` rejects out-of-range; max uses `uint32(ndg2Chunk+Overhead()+64)` after int add. |
| 22 | `nvr/retention.go` `int64(Bavail)*int64(Bsize)` | real-ish | Bavail is **uint64** on linux amd64. **Fixed**: overflow-checked multiply. |
| 23–24 | `nvr/storage_status.go` `Bavail*uint64(Bsize)` | real-ish | Bsize is **int64**. **Fixed**: float path `float64(Bavail)*float64(Bsize)`. |
| 25 | `revbridge/ws.go` `int(Uint64(ext))` then `make([]byte, n)` | **real** | Peer-controlled length, **no cap** → DoS. **Fixed**: reject > 64MiB before int cast and alloc. |
| 26 | `revbridge/ws.go` `byte(n>>8)` in write | protocol | n already chunked ≤ 32KiB in Write(). **Leave**. |
| 27 | `revbridge/mux.go` `uint32(len(payload))` | safe-bound | `len > maxFrame` (1MiB) returns error first. **Leave** (comment). |

## Tapo package (protocol crypto)

| Area | Findings | Decision |
|--|--|--|
| `ccm.go` | int→byte for L, tagSize, length fields | **protocol** — CCM L∈[2,8], tag fixed, payload sizes for Tapo frames. **Leave**; not network-generic codec. |
| `klap.go` | int32↔uint32 seq | **protocol** — KLAP sequence wire format. **Leave**. |
| `tpap.go` | EC y parity byte; seq uint32 | **protocol**. **Leave**. |
| `client.go` | PKCS7 `byte(pad)` | pad ∈ 1..block. **Leave**. |

Revisit only if Tapo code is reused as a general crypto helper.

## Residual G115 expected after this pass

- mux `uint32(len)` after maxFrame check  
- ws write length bytes  
- tapo protocol casts  
- Possibly still backup/overhead depending on gosec path sensitivity  

These are **documented safe-bound / protocol**, not open P0.

## Verification

```text
go test -count=1 ./internal/revbridge/ ./internal/nvr/ ./internal/install/ \
  ./internal/mtls/ ./internal/secondary/ ./internal/svcpaths/ ./internal/firewall/
# all ok (2026-11-18 sandbox)
```

## Follow-ups (not this pass)

- Re-run **ai-tools ≥ 1.12.1** / gosec and attach delta issue count.  
- N3 residual **U1000** in internal deploy/firewall/svcpaths: wire-or-delete audit (separate journal).  
- No production release until operator smoke after apply.


## Re-run 2026-11-18 (post-fix tip `9205bbe`)

Tool: **ai-tools v1.12.1** · report artifacts: `aitools-report-netductor-post-g115.json`

| Metric | Before (v1121 / pre-fix) | After |
|--|--|--|
| gosec total | 1213 | 1186 |
| **G115** | **43** | **20** |
| G123 (mTLS) | 1 (pre-N2) | **0** |

### G115 cleared (27 sites)

All **FP-uid** euid sites, firewall hostU32, backup PutUint32/old Overhead form, nvr Statfs old form, **ws read Uint64→int**, mux old line, secondary/store, svcpaths peer/bootstrap.

### G115 remaining (20)

| Sites | Class | Note |
|--|--|--|
| **tapo/** (16) | protocol | CCM / KLAP / TPAP / PKCS7 — leave |
| `install/backup.go:274` | safe-bound | `ctLen > uint32(ndg2Chunk+Overhead()+64)` — chunk cap |
| `nvr/retention.go:84` | safe-bound | final `int64(Bavail*bsize)` after overflow check |
| `revbridge/mux.go:94` | safe-bound | `len ≤ maxFrame` (1MiB) before `uint32` |
| `revbridge/ws.go:177` | protocol | `byte(n>>8)` in write; Write() splits ≤32KiB |

No further code change required for residual G115 unless product policy demands nosec/helpers for the four non-tapo lines.

### Other tools this run

- staticcheck / errcheck / revive / gofumpt / ineffassign: completed (findings not re-triaged here)
- deadcode / govulncheck: killed (OOM) in sandbox — not used as evidence
- shellcheck: completed

