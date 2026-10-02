# Deep code review — 2026-10-02 (stop line ~v0.9.191)

**Scope:** repository `PavelNeyman/netductor`, ~345 `.go` files, ~52k LOC (excl. tests), 82 test files.  
**Method:** full `go test ./...` (all packages with tests **PASS**), structured read of auth/API/edge/VPN/install/stack/backup/operator paths, security pattern scans, gap analysis for untested packages.  
**Not claimed:** line-by-line human audit of every TG handler string; live VPS proof.

---

## 1. Executive summary

| Area | Verdict |
|------|---------|
| Build / unit tests | **Green** — `go test ./...` ok |
| Architecture freeze alignment | **Mostly strong** — loopback API, mTLS agent plane, Mac op UI, backup_pull not SCP, stack no auto-downgrade |
| Security posture | **Solid core** with known residual (legacy sessions, InsecureSkipVerify on recovery/LAN, guest PIN length-CT, git `rev` not fully constrained) |
| OpenWrt path | **Coherent after 0.9.186–191**; residual radio0 / iface naming |
| Coverage holes | **No tests:** `svcpaths`, `tlsle`, `dnsblock`, `fleet`, `addons`, `hardening`, `cmd/netductor-op` |
| Production readiness | **Code-ready for dual-VPS + edge design**; hardware/live matrix still required |

---

## 2. Test reality

```
ok  cmd/netductor, agent, tg, deploy, edge, edgeagent, install, stack, vpn, nvr, …
?   cmd/netductor-op, internal/svcpaths, tlsle, dnsblock, fleet, addons, hardening, version
```

Largest production files (review priority): `operator/serve.go` (~929), `tg/main.go` (~918), `install/backup.go` (~790), `edge/store.go` (~705), `stack/stack.go` (~596), `secondary/agent.go` (~631).

---

## 3. Security findings (detailed)

### 3.1 Strengths

| Control | Implementation |
|---------|----------------|
| Node API bind | Only `127.0.0.1` / `localhost`; non-local exit 2 (`cmd/netductor/serve.go`) |
| VPS admin UI | Removed; `/admin` → 410 |
| Agent plane | mTLS TLS1.3 (`internal/mtls`); plain :8788 removed from product path |
| Session tokens | 256-bit random; **hashed** on disk (`session.Create` / `hashToken`) |
| Edge / secondary tokens | `subtle.ConstantTimeCompare` on SHA-256 digests (`edge.constEq`, secondary store) |
| Operator Web token | `safeOperatorToken` charset + length; constant-time compare on op serve |
| Systemd restart API | Allowlist units (`sing-box`, api, blocky, tg) |
| Guest internet | Zone forward REJECT + nft MAC set; not open WAN forward |
| Backup perms | Secrets often `0o600`; dirs `0o700` |
| Sub public endpoint | Rate limit present + tests (`redirect_test.go`) |

### 3.2 Issues / residual risk

| ID | Severity | Location | Issue | Recommendation |
|----|----------|----------|-------|----------------|
| **S1** | Medium | `internal/session` | Legacy session files named by **plaintext token** still loaded | Drop legacy path after migration window; force revoke-all on upgrade |
| **S2** | Medium | `internal/git.Show` | `rev` passed to `git show` without charset allowlist | Restrict to `[A-Za-z0-9._/-]` or resolve via `git rev-parse --verify` |
| **S3** | Low–Med | `guest.PINOK` | Plain `DeskPIN` path: `ConstantTimeCompare` on unequal lengths → always fail (ok) but not constant-time vs length oracle | Prefer hash-only PIN storage |
| **S4** | Low (accepted) | tapo, probes, backup recover, sni_health | `InsecureSkipVerify` | Document; pin certs where possible for recovery |
| **S5** | Low | `svcpaths` | `bash -c` with `shellQuote` for wg pubkey | Prefer `wg pubkey` via stdin without shell |
| **S6** | Ops | Edge harden | Harden failure is **warn**, deploy continues | Optional hard-fail flag |
| **S7** | Ops | OpenWrt | Guest `radio0` only; `default_radio*` names | Probe wireless; multi-radio |
| **S8** | Info | Defaults | SNI default `api.vk.me` | Already overridable; keep explicit in wizard |

### 3.3 Auth model map

```
Mac op (localhost Web/TUI) --SSH tunnel / VPN--> primary :8787 loopback + session/Bearer
Edge/Secondary agents ----mTLS :8789----------> primary agent plane
Public users -------------VLESS :443-----------> secondary (entry)
Redirect/sub -------------:8443/80------------> tokenized links only
```

---

## 4. Module-by-module notes

### 4.1 `cmd/netductor` (node)

- Mux composition clear; NVR background start on serve.
- `requireSession` → `session.Valid` — good default for mutating APIs (verify each register* uses it — pattern is consistent in session/sni handlers reviewed).
- Doctor/collect shell out to systemctl — expected for node role.

### 4.2 `internal/deploy` + `edgeagent`

- First-boot order correct post-0.9.189.
- `ShellApplyStaged` ash-safe post-0.9.191.
- SSH prefers password when set; Dropbear-oriented transfer history sound.
- **Gap:** no automated test for full shell script execution under `ash -n`.

### 4.3 `cmd/netductor-agent`

- Enroll backoff, guest stage, LuCI TTL files, VPN init.d generation — coherent.
- NVR on-router path exists; **flash/RAM** risk on Cudy if recording local.
- Recovery HTTP LAN-bound design present.

### 4.4 `internal/edge` / `vpn`

- Enroll rate limit + tests.
- `edge-*` peer naming + user list filter.
- TemplateWithVPN policy merge tests exist.

### 4.5 `internal/install` / `stack` / backup

- COMPONENT-driven install; LE reissue helpers.
- backup_pull only offsite path; SCP peer **removed**.
- Stack: lock, systemd-run apply, **no auto version downgrade** on health fail (explicit product decision after 0.9.121 incidents).

### 4.6 `internal/operator` (Mac)

- Local bind default; token injection into SPA.
- Fleet/primary/secondary/edge handlers — large surface; **no unit tests on cmd/netductor-op**.
- Highest residual **regression risk** for deploy wizards (password/domain/addons selection bugs historically).

### 4.7 `cmd/netductor-tg`

- Large callback surface (`handlers_cb`, keyboards).
- Tests exist at package level but **not** full callback matrix coverage.
- Format helpers for stack/status — improved; screen-by-screen template compliance not fully machine-checked.

### 4.8 `internal/nvr` / `tapo` / `mikrotik`

- PathUnderRoot prevents trivial `..` escape after Clean (symlink-follow edge case remains OS-dependent).
- Retention tests exist.
- Tapo KLAP + InsecureSkipVerify for LAN cameras — expected.
- MikroTik RSC generation + SSH — no live proof.

### 4.9 `internal/svcpaths`

- **No tests**; shell-heavy bootstrap; critical for SP/PS — prioritize tests + reduce `bash -c`.

### 4.10 Git / CI / registry

- Repo name sanitize alphanumeric — good.
- Pipeline hook embeds sanitized name — ok if sanitize strict.
- `git show` rev — tighten (S2).

---

## 5. Reliability / ops findings

| Topic | Note |
|-------|------|
| Version display | Historical TG “0.9.121” was stack prev/UI; policy now keeps binaries on failed health |
| Recover | Two-pass install; LE outside archive |
| Secondary | Not full mirror — agent + VPN entry |
| Addons | Must stay opt-in in Fleet (past bug: silent install) |

---

## 6. Recommended fix order (code)

1. **P0** — Remove or time-box legacy session plaintext files (S1).  
2. **P0** — Sanitize `git Show` rev (S2).  
3. **P1** — Guest multi-radio probe (S7).  
4. **P1** — Unit tests for `svcpaths` key material + idempotent apply.  
5. **P2** — DeskPIN hash-only; optional harden hard-fail; `ash -n` test for ShellApplyStaged.  
6. **P2** — Expand op deploy golden tests (addons flags, domain fields).

---

## 7. What “deep” still cannot replace

- Live dual-node smoke (Phase C).  
- OpenWrt Cudy e2e (Phase D).  
- Camera/MikroTik (Phase F).  
- Client DPI/white-list behavior (Phase E).

Use `scripts/collect-vps-state.sh` on each VPS and attach output for offline assessment.

---

## 8. Confidence

| Layer | Confidence |
|-------|------------|
| Security critical paths reviewed | **High** |
| Entire TG/op UI correctness | **Medium** (structure ok, not exhaustive) |
| Untested packages behavior | **Medium-Low** until tests added |
| Hardware behavior | **None** without reports |

